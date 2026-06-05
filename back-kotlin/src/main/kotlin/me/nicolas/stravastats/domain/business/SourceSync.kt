package me.nicolas.stravastats.domain.business

data class SourceSyncResult(
    val status: String,
    val reason: String,
    val message: String,
    val startedAt: String = "",
    val completedAt: String = "",
    val durationMs: Long = 0,
    val reloaded: Boolean = false,
    val fit: FITImportResult = FITImportResult(),
)

data class FITImportResult(
    val status: String = "idle",
    val message: String = "FIT import has not run yet.",
    val configured: Boolean = false,
    val sourceKind: String = "",
    val sourcePath: String = "",
    val inboxPath: String = "",
    val candidateSourcePaths: List<String> = emptyList(),
    val destinationPath: String = "",
    val scannedFiles: Int = 0,
    val importedFiles: Int = 0,
    val alreadyPresentFiles: Int = 0,
    val skippedFiles: Int = 0,
    val invalidFiles: Int = 0,
    val createdYearDirectories: List<String> = emptyList(),
    val imported: List<ImportedFITFile> = emptyList(),
    val errors: List<String> = emptyList(),
    val deviceSync: FITDeviceSyncResult? = null,
)

data class ImportedFITFile(
    val source: String,
    val destination: String,
    val year: String,
    val activityId: Long,
    val startDate: String,
)

data class FITDeviceSyncResult(
    val status: String,
    val message: String,
    val backend: String = "filesystem",
    val device: String = "",
    val sourcePath: String = "",
    val inboxPath: String = "",
    val candidateSourcePaths: List<String> = emptyList(),
    val scannedFiles: Int = 0,
    val copiedFiles: Int = 0,
    val alreadyPresentFiles: Int = 0,
    val skippedFiles: Int = 0,
    val invalidFiles: Int = 0,
    val copied: List<FITDeviceSyncFile> = emptyList(),
    val errors: List<String> = emptyList(),
)

data class FITDeviceSyncFile(
    val source: String,
    val destination: String,
)
