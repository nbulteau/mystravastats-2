package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.adapters.localrepositories.fit.FITRepository
import me.nicolas.stravastats.domain.RuntimeConfig
import me.nicolas.stravastats.domain.business.FITDeviceSyncFile
import me.nicolas.stravastats.domain.business.FITDeviceSyncResult
import me.nicolas.stravastats.domain.business.FITImportResult
import me.nicolas.stravastats.domain.business.ImportedFITFile
import me.nicolas.stravastats.domain.business.SourceSyncResult
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.slf4j.LoggerFactory
import org.springframework.boot.context.event.ApplicationReadyEvent
import org.springframework.context.event.EventListener
import org.springframework.stereotype.Service
import java.io.File
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import java.time.Duration
import java.time.Instant
import java.time.LocalDateTime
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.concurrent.thread
import kotlin.math.roundToLong

interface ISourceSyncService {
    fun synchronize(reason: String = "manual"): SourceSyncResult
    fun lastResult(): SourceSyncResult
}

@Service
class SourceSyncService(
    private val activityProvider: IActivityProvider,
) : ISourceSyncService {
    private val logger = LoggerFactory.getLogger(SourceSyncService::class.java)
    private val running = AtomicBoolean(false)

    @Volatile
    private var lastResult = SourceSyncResult(
        status = "idle",
        reason = "",
        message = "Synchronization has not run yet.",
        fit = FITImportResult(status = "idle", message = "FIT import has not run yet."),
    )

    override fun synchronize(reason: String): SourceSyncResult {
        val normalizedReason = reason.trim().ifEmpty { "manual" }
        if (!running.compareAndSet(false, true)) {
            return lastResult.copy(
                status = "running",
                reason = normalizedReason,
                message = "Synchronization is already running.",
            )
        }

        return try {
            val started = Instant.now()
            val fit = importFIT()
            val reloaded = if (fit.importedFiles > 0) activityProvider.reload() else false
            val completed = Instant.now()
            SourceSyncResult(
                status = syncStatusFromFIT(fit),
                reason = normalizedReason,
                message = syncMessageFromFIT(fit),
                startedAt = started.toString(),
                completedAt = completed.toString(),
                durationMs = Duration.between(started, completed).toMillis(),
                reloaded = reloaded,
                fit = fit,
            ).also { result -> lastResult = result }
        } finally {
            running.set(false)
        }
    }

    override fun lastResult(): SourceSyncResult = lastResult

    @EventListener(ApplicationReadyEvent::class)
    fun synchronizeOnStartup() {
        thread(name = "source-sync-startup", isDaemon = true) {
            synchronize("startup")
        }
    }

    private fun importFIT(): FITImportResult {
        val destinationPath = RuntimeConfig.readConfigValue("FIT_FILES_PATH").orEmpty().trim()
        val inboxPath = RuntimeConfig.fitInboxPath().first.trim()
        if (destinationPath.isEmpty()) {
            return FITImportResult(
                status = "not_configured",
                message = "FIT directory is not configured.",
                configured = false,
                destinationPath = destinationPath,
                inboxPath = inboxPath,
            )
        }

        val errors = mutableListOf<String>()
        if (inboxPath.isNotEmpty()) {
            try {
                File(inboxPath).mkdirs()
            } catch (exception: Exception) {
                return FITImportResult(
                    status = "failed",
                    message = "Unable to create FIT inbox directory.",
                    configured = true,
                    destinationPath = destinationPath,
                    inboxPath = inboxPath,
                    errors = listOf(exception.message ?: exception.toString()),
                )
            }
        }

        val deviceSync = syncGarminToInbox(inboxPath)
        if (deviceSync?.status == "failed") {
            errors += deviceSync.errors
        }

        val (sourcePath, sourceKind, candidates) = detectFITImportSource(inboxPath)
        if (sourcePath.isEmpty()) {
            return FITImportResult(
                status = "no_device",
                message = "No FIT inbox or Garmin USB activity directory was detected.",
                configured = true,
                destinationPath = destinationPath,
                inboxPath = inboxPath,
                candidateSourcePaths = (deviceSync?.candidateSourcePaths.orEmpty() + candidates).distinct(),
                errors = errors,
                deviceSync = deviceSync,
            )
        }

        val destinationDirectory = File(destinationPath)
        if (!destinationDirectory.exists() && !destinationDirectory.mkdirs()) {
            return FITImportResult(
                status = "failed",
                message = "Unable to create FIT destination directory.",
                configured = true,
                sourceKind = sourceKind,
                sourcePath = sourcePath,
                inboxPath = inboxPath,
                candidateSourcePaths = (deviceSync?.candidateSourcePaths.orEmpty() + candidates).distinct(),
                destinationPath = destinationPath,
                errors = errors + "Unable to create $destinationPath",
                deviceSync = deviceSync,
            )
        }

        val decoder = FITRepository(destinationPath)
        val existingFingerprints = existingFITFingerprints(destinationDirectory, File(sourcePath), decoder).toMutableSet()
        val imported = mutableListOf<ImportedFITFile>()
        val createdYears = mutableSetOf<String>()
        var scannedFiles = 0
        var importedFiles = 0
        var alreadyPresentFiles = 0
        var skippedFiles = 0
        var invalidFiles = 0

        fitFiles(File(sourcePath)).forEach { file ->
            scannedFiles++
            val activity = decoder.decodeActivity(file)
            if (activity == null) {
                invalidFiles++
                errors += "${file.absolutePath}: unable to decode FIT file"
                return@forEach
            }
            val fingerprint = activityFingerprint(activity)
            if (fingerprint.isNotEmpty() && existingFingerprints.contains(fingerprint)) {
                alreadyPresentFiles++
                skippedFiles++
                return@forEach
            }

            val year = activityYear(activity)
            val yearDirectory = File(destinationDirectory, year)
            if (!yearDirectory.exists()) {
                createdYears += year
            }
            if (!yearDirectory.exists() && !yearDirectory.mkdirs()) {
                invalidFiles++
                errors += "Unable to create ${yearDirectory.absolutePath}"
                return@forEach
            }

            val destination = destinationFilePath(file, yearDirectory)
            try {
                copyFileAtomic(file, destination)
                importedFiles++
                if (fingerprint.isNotEmpty()) {
                    existingFingerprints += fingerprint
                }
                if (imported.size < 25) {
                    imported += ImportedFITFile(
                        source = file.absolutePath,
                        destination = destination.absolutePath,
                        year = year,
                        activityId = activity.id,
                        startDate = activity.startDateLocal.ifEmpty { activity.startDate },
                    )
                }
            } catch (exception: Exception) {
                invalidFiles++
                errors += exception.message ?: exception.toString()
            }
        }

        val status = fitImportStatus(scannedFiles, importedFiles, invalidFiles)
        if (errors.isNotEmpty()) {
            logger.warn("FIT import completed with {} error(s)", errors.size)
        }
        return FITImportResult(
            status = status,
            message = fitImportMessage(status, sourceKind, importedFiles, alreadyPresentFiles),
            configured = true,
            sourceKind = sourceKind,
            sourcePath = sourcePath,
            inboxPath = inboxPath,
            candidateSourcePaths = (deviceSync?.candidateSourcePaths.orEmpty() + candidates).distinct(),
            destinationPath = destinationPath,
            scannedFiles = scannedFiles,
            importedFiles = importedFiles,
            alreadyPresentFiles = alreadyPresentFiles,
            skippedFiles = skippedFiles,
            invalidFiles = invalidFiles,
            createdYearDirectories = createdYears.sorted(),
            imported = imported,
            errors = errors,
            deviceSync = deviceSync,
        )
    }

    private fun syncGarminToInbox(inboxPath: String): FITDeviceSyncResult? {
        if (inboxPath.isBlank()) return null
        val (device, candidates) = detectGarminDevice()
        if (device == null) {
            return FITDeviceSyncResult(
                status = "no_device",
                message = "No mounted Garmin activity directory was detected.",
                inboxPath = inboxPath,
                candidateSourcePaths = candidates,
            )
        }

        val inbox = File(inboxPath)
        if (!inbox.exists() && !inbox.mkdirs()) {
            return FITDeviceSyncResult(
                status = "failed",
                message = "Unable to create FIT inbox directory.",
                device = device.name,
                sourcePath = device.activityPath.absolutePath,
                inboxPath = inboxPath,
                candidateSourcePaths = candidates,
                errors = listOf("Unable to create $inboxPath"),
            )
        }

        val copied = mutableListOf<FITDeviceSyncFile>()
        val errors = mutableListOf<String>()
        var copiedFiles = 0
        var alreadyPresentFiles = 0
        var skippedFiles = 0
        var invalidFiles = 0
        val files = fitFiles(device.activityPath)
        files.forEach { file ->
            try {
                val (destination, wasCopied) = copyFITToInbox(file, inbox)
                if (wasCopied) {
                    copiedFiles++
                    if (copied.size < 25) {
                        copied += FITDeviceSyncFile(file.absolutePath, destination.absolutePath)
                    }
                } else {
                    alreadyPresentFiles++
                    skippedFiles++
                }
            } catch (exception: Exception) {
                invalidFiles++
                errors += "${file.absolutePath}: ${exception.message ?: exception}"
            }
        }

        val result = FITDeviceSyncResult(
            status = "ok",
            message = "",
            device = device.name,
            sourcePath = device.activityPath.absolutePath,
            inboxPath = inboxPath,
            candidateSourcePaths = candidates,
            scannedFiles = files.size,
            copiedFiles = copiedFiles,
            alreadyPresentFiles = alreadyPresentFiles,
            skippedFiles = skippedFiles,
            invalidFiles = invalidFiles,
            copied = copied,
            errors = errors,
        )
        return result.copy(message = garminDeviceSyncMessage(result))
    }

    private fun detectFITImportSource(inboxPath: String): Triple<String, String, List<String>> {
        val candidates = mutableListOf<String>()
        if (inboxPath.isNotBlank()) {
            val inbox = File(inboxPath)
            candidates += inbox.absolutePath
            if (inbox.isDirectory) {
                return Triple(inbox.absolutePath, "fit_inbox", candidates)
            }
        }
        val (device, garminCandidates) = detectGarminDevice()
        candidates += garminCandidates
        if (device != null) {
            return Triple(device.activityPath.absolutePath, "garmin_usb", candidates)
        }
        return Triple("", "", candidates)
    }

    private fun detectGarminDevice(): Pair<GarminDevice?, List<String>> {
        val candidates = mutableListOf<File>()
        val configuredSource = RuntimeConfig.readConfigValue("GARMIN_FIT_SOURCE_PATH")
        if (!configuredSource.isNullOrBlank()) {
            appendGarminSourceCandidates(File(configuredSource), candidates)
        } else {
            platformVolumeRoots().forEach { root -> appendGarminVolumeCandidates(root, candidates) }
        }
        val distinctCandidates = candidates.map { it.absoluteFile.normalize() }.distinctBy { it.absolutePath }
        val device = distinctCandidates.firstOrNull { it.isDirectory }?.let { activityPath ->
            GarminDevice(
                name = garminDeviceName(activityPath),
                root = garminDeviceRoot(activityPath),
                activityPath = activityPath,
            )
        }
        return device to distinctCandidates.map { it.absolutePath }
    }

    private fun existingFITFingerprints(destinationDirectory: File, sourceDirectory: File, decoder: FITRepository): Set<String> {
        return fitFiles(destinationDirectory)
            .filterNot { file -> file.absolutePath == sourceDirectory.absolutePath || file.isInside(sourceDirectory) }
            .mapNotNull { file -> decoder.decodeActivity(file)?.let(::activityFingerprint)?.takeIf { it.isNotEmpty() } }
            .toSet()
    }

    private fun fitFiles(root: File): List<File> {
        if (!root.isDirectory) return emptyList()
        return root.walkTopDown()
            .filter { file -> file.isFile && file.extension.equals("fit", ignoreCase = true) }
            .sortedBy { file -> file.absolutePath }
            .toList()
    }

    private fun copyFITToInbox(source: File, inbox: File): Pair<File, Boolean> {
        val preferred = File(inbox, source.name.ifBlank { "activity.fit" })
        if (sameFileSize(source, preferred)) {
            return preferred to false
        }
        val destination = availableInboxDestination(source, preferred)
        copyFileAtomic(source, destination)
        return destination to true
    }

    private fun sameFileSize(source: File, destination: File): Boolean {
        return source.isFile && destination.isFile && source.length() == destination.length()
    }

    private fun availableInboxDestination(source: File, preferred: File): File {
        if (!preferred.exists()) return preferred
        val extension = preferred.extension.ifBlank { "fit" }
        val stem = preferred.nameWithoutExtension.ifBlank { "activity" }
        val sourceSize = source.length()
        for (index in 1 until 1000) {
            val candidate = File(preferred.parentFile, "$stem-$sourceSize-$index.$extension")
            if (!candidate.exists()) return candidate
        }
        error("unable to choose destination file for ${source.absolutePath}")
    }

    private fun destinationFilePath(source: File, yearDirectory: File): File {
        val preferred = File(yearDirectory, source.name.ifBlank { "activity.fit" })
        if (!preferred.exists()) return preferred
        val hash = shortFileHash(source)
        val extension = preferred.extension.takeIf { it.isNotBlank() }?.let { ".$it" }.orEmpty()
        val stem = preferred.nameWithoutExtension.ifBlank { "activity" }
        for (index in 0 until 100) {
            val suffix = if (index == 0) hash else "$hash-$index"
            val candidate = File(yearDirectory, "$stem-$suffix$extension")
            if (!candidate.exists()) return candidate
        }
        error("unable to choose destination file for ${source.absolutePath}")
    }

    private fun copyFileAtomic(source: File, destination: File) {
        val temp = File(destination.parentFile, "${destination.name}.tmp")
        Files.copy(source.toPath(), temp.toPath(), StandardCopyOption.REPLACE_EXISTING)
        Files.move(temp.toPath(), destination.toPath(), StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE)
    }

    private fun shortFileHash(file: File): String {
        val digest = java.security.MessageDigest.getInstance("SHA-256").digest(file.readBytes())
        return digest.joinToString("") { byte -> "%02x".format(byte) }.take(10)
    }

    private fun activityFingerprint(activity: StravaActivity): String {
        val start = parseActivityDate(activity.startDate.ifBlank { activity.startDateLocal }) ?: return ""
        return listOf(
            activity.type.trim(),
            start.toString(),
            activity.distance.roundToLong().toString(),
            activity.elapsedTime.toString(),
        ).joinToString("|")
    }

    private fun activityYear(activity: StravaActivity): String {
        listOf(activity.startDateLocal, activity.startDate).forEach { value ->
            parseActivityDate(value)?.let { return it.atZone(ZoneId.systemDefault()).year.toString() }
            if (value.trim().length >= 4 && value.trim().take(4).all(Char::isDigit)) {
                return value.trim().take(4)
            }
        }
        return LocalDateTime.now().year.toString()
    }

    private fun parseActivityDate(value: String): Instant? {
        val raw = value.trim()
        if (raw.isEmpty()) return null
        return runCatching { Instant.parse(raw) }.getOrNull()
            ?: runCatching { OffsetDateTime.parse(raw).toInstant() }.getOrNull()
            ?: runCatching {
                LocalDateTime.parse(raw, DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.ROOT))
                    .atZone(ZoneId.systemDefault())
                    .toInstant()
            }.getOrNull()
            ?: runCatching { LocalDateTime.parse(raw).atZone(ZoneId.systemDefault()).toInstant() }.getOrNull()
    }

    private fun syncStatusFromFIT(fit: FITImportResult): String {
        return when (fit.status) {
            "failed" -> "failed"
            "not_configured", "no_device" -> "skipped"
            else -> "completed"
        }
    }

    private fun syncMessageFromFIT(fit: FITImportResult): String {
        return when (fit.status) {
            "imported" -> "Imported ${fit.importedFiles} FIT file(s)."
            "up_to_date" -> "FIT library is already up to date."
            "no_files" -> if (fit.sourceKind == "fit_inbox") {
                "FIT inbox is configured, but no FIT files were present."
            } else {
                "Garmin USB source was found, but no FIT files were present."
            }
            "no_device" -> "No FIT inbox or Garmin USB activity directory was detected."
            "not_configured" -> "FIT import skipped because FIT_FILES_PATH is not configured."
            "failed" -> "FIT import failed."
            else -> fit.message
        }
    }

    private fun fitImportStatus(scannedFiles: Int, importedFiles: Int, invalidFiles: Int): String {
        if (scannedFiles == 0) return "no_files"
        if (importedFiles > 0) return "imported"
        if (invalidFiles > 0 && invalidFiles == scannedFiles) return "failed"
        return "up_to_date"
    }

    private fun fitImportMessage(status: String, sourceKind: String, importedFiles: Int, alreadyPresentFiles: Int): String {
        return when (status) {
            "imported" -> "$importedFiles new FIT file(s) imported into year folders."
            "up_to_date" -> "$alreadyPresentFiles FIT file(s) already present."
            "no_files" -> if (sourceKind == "fit_inbox") {
                "No FIT files found in the configured FIT inbox."
            } else {
                "No FIT files found in the detected Garmin activity directory."
            }
            "failed" -> "No valid FIT file could be imported."
            else -> ""
        }
    }

    private fun garminDeviceSyncMessage(result: FITDeviceSyncResult): String {
        if (result.status == "failed") return "Garmin FIT synchronization failed."
        if (result.copiedFiles > 0) return "Copied ${result.copiedFiles} FIT file(s) from Garmin source to inbox."
        if (result.scannedFiles == 0) return "Mounted Garmin activity directory was found, but it contains no FIT files."
        return "${result.alreadyPresentFiles} FIT file(s) already present in inbox."
    }

    private fun appendGarminSourceCandidates(source: File, candidates: MutableList<File>) {
        appendGarminActivitySubdirectories(source, candidates)
        candidates += source
    }

    private fun appendGarminVolumeCandidates(root: File, candidates: MutableList<File>) {
        root.listFiles()
            ?.filter { file -> file.isDirectory && !file.name.startsWith(".") }
            ?.forEach { volume -> appendGarminActivitySubdirectories(volume, candidates) }
    }

    private fun appendGarminActivitySubdirectories(root: File, candidates: MutableList<File>) {
        listOf(
            listOf("GARMIN", "ACTIVITY"),
            listOf("GARMIN", "Activity"),
            listOf("Garmin", "ACTIVITY"),
            listOf("Garmin", "Activity"),
        ).forEach { parts -> candidates += File(File(root, parts[0]), parts[1]) }
    }

    private fun platformVolumeRoots(): List<File> {
        val osName = System.getProperty("os.name").lowercase(Locale.ROOT)
        return when {
            osName.contains("mac") -> listOf(File("/Volumes"))
            osName.contains("win") -> ('A'..'Z').map { letter -> File("$letter:\\") }
            osName.contains("linux") -> {
                val user = System.getenv("USER").orEmpty().trim()
                buildList {
                    if (user.isNotEmpty()) {
                        add(File("/run/media/$user"))
                        add(File("/media/$user"))
                    }
                    add(File("/media"))
                    add(File("/mnt"))
                }
            }
            else -> emptyList()
        }
    }

    private fun garminDeviceName(activityPath: File): String {
        return garminDeviceRoot(activityPath).name.ifBlank { "Garmin" }
    }

    private fun garminDeviceRoot(activityPath: File): File {
        return activityPath.parentFile?.parentFile ?: activityPath
    }

    private fun File.isInside(root: File): Boolean {
        val relative = runCatching { root.toPath().normalize().relativize(this.toPath().normalize()) }.getOrNull()
        return relative != null && relative.toString().isNotBlank() && !relative.startsWith("..")
    }

    private fun File.normalize(): File = toPath().normalize().toFile()

    private data class GarminDevice(
        val name: String,
        val root: File,
        val activityPath: File,
    )
}
