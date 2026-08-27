package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.utils.SafeLocalFile
import org.springframework.stereotype.Service
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.io.File
import java.time.Instant

data class LocalDataBackup(
    val version: Int,
    val exportedAt: String,
    val athleteId: String,
    val files: Map<String, Any?>,
)

data class LocalDataRestoreResult(
    val restored: List<String>,
)

interface ILocalDataBackupService {
    fun export(): LocalDataBackup
    fun restore(backup: LocalDataBackup): LocalDataRestoreResult
}

@Service
class LocalDataBackupService(
    private val activityProvider: IActivityProvider,
) : ILocalDataBackupService {
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .build()

    override fun export(): LocalDataBackup {
        val identity = requireNotNull(activityProvider.cacheIdentity()) { "Local data backup is unavailable for this provider" }
        val directory = File(identity.cacheRoot, "strava-${identity.athleteId}")
        val files = linkedMapOf<String, Any?>()
        localFileNames.forEach { (key, fileName) ->
            val file = File(directory, fileName(identity.athleteId))
            if (file.isFile) {
                files[key] = objectMapper.readValue(file, Any::class.java)
            }
        }
        return LocalDataBackup(
            version = LOCAL_DATA_BACKUP_VERSION,
            exportedAt = Instant.now().toString(),
            athleteId = identity.athleteId,
            files = files,
        )
    }

    override fun restore(backup: LocalDataBackup): LocalDataRestoreResult {
        val identity = requireNotNull(activityProvider.cacheIdentity()) { "Local data restore is unavailable for this provider" }
        require(backup.version == LOCAL_DATA_BACKUP_VERSION) { "Unsupported backup version ${backup.version}" }
        require(backup.athleteId == identity.athleteId) {
            "Backup belongs to athlete '${backup.athleteId}', current athlete is '${identity.athleteId}'"
        }
        val unsupportedKeys = backup.files.keys - localFileNames.keys
        require(unsupportedKeys.isEmpty()) { "Unsupported local data keys: ${unsupportedKeys.sorted().joinToString()}" }

        // Serialize and validate every entry before replacing any current file.
        val serialized = backup.files.mapValues { (_, value) -> objectMapper.writeValueAsBytes(value) }
        val directory = File(identity.cacheRoot, "strava-${identity.athleteId}")
        val restored = serialized.keys.sorted()
        restored.forEach { key ->
            val file = File(directory, localFileNames.getValue(key)(identity.athleteId))
            SafeLocalFile.withLock(file) {
                SafeLocalFile.write(file, serialized.getValue(key))
            }
        }
        return LocalDataRestoreResult(restored)
    }
}

private const val LOCAL_DATA_BACKUP_VERSION = 1

private val localFileNames = mapOf<String, (String) -> String>(
    "dataQualityCorrections" to { id -> "data-quality-corrections-$id.json" },
    "dataQualityExclusions" to { id -> "data-quality-exclusions-$id.json" },
    "gearMaintenance" to { id -> "gear-maintenance-$id.json" },
    "heartRateZones" to { id -> "heart-rate-zones-$id.json" },
    "performanceSettings" to { id -> "performance-settings-$id.json" },
)
