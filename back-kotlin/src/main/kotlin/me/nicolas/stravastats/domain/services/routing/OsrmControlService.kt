package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.RuntimeConfig
import org.springframework.http.HttpStatus
import org.springframework.stereotype.Service
import org.springframework.web.server.ResponseStatusException
import java.io.File
import java.util.Locale
import java.util.concurrent.CompletableFuture
import java.util.concurrent.TimeUnit

private const val DEFAULT_OSRM_CONTROL_COMPOSE_FILE = "docker-compose-routing-osrm.yml"
private const val DEFAULT_OSRM_CONTROL_TIMEOUT_MS = 30_000
private const val MAX_OSRM_CONTROL_OUTPUT_CHARS = 4_000

data class OsrmControlResult(
    val status: String,
    val message: String,
    val command: String,
    val projectDir: String,
    val composeFile: String,
    val output: String = "",
)

interface IOsrmControlService {
    fun startOsrm(): OsrmControlResult
}

@Service
class OsrmControlService : IOsrmControlService {

    override fun startOsrm(): OsrmControlResult {
        val projectDir = resolveProjectDir()
        val composeFile = resolveComposeFile(projectDir)
        val dockerBin = resolveDockerBinary()
        val command = listOf(dockerBin ?: "docker", "compose", "-f", composeFile.absolutePath, "up", "-d", "osrm")
        val unavailable = OsrmControlResult(
            status = "unavailable",
            message = "OSRM start command was not run.",
            command = commandDisplay(command),
            projectDir = projectDir.absolutePath,
            composeFile = composeFile.absolutePath,
        )

        if (!readBoolConfig("OSRM_CONTROL_ENABLED", true)) {
            throw ResponseStatusException(
                HttpStatus.FORBIDDEN,
                "OSRM control is disabled. Set OSRM_CONTROL_ENABLED=true to allow starting OSRM from the UI.",
            )
        }
        if (!composeFile.isFile) {
            throw ResponseStatusException(HttpStatus.CONFLICT, "OSRM compose file not found: ${composeFile.absolutePath}")
        }
        if (dockerBin == null) {
            throw ResponseStatusException(
                HttpStatus.CONFLICT,
                "Docker CLI not found. Install Docker Desktop or set OSRM_CONTROL_DOCKER_BIN.",
            )
        }

        val process = ProcessBuilder(command)
            .directory(projectDir)
            .redirectErrorStream(true)
            .start()
        val outputFuture = CompletableFuture.supplyAsync {
            process.inputStream.bufferedReader().use { it.readText() }
        }

        val timeoutMs = osrmControlTimeoutMs()
        val completed = process.waitFor(timeoutMs.toLong(), TimeUnit.MILLISECONDS)
        if (!completed) {
            process.destroyForcibly()
            throw ResponseStatusException(
                HttpStatus.GATEWAY_TIMEOUT,
                "OSRM start command timed out after ${timeoutMs}ms.",
            )
        }

        val output = trimOutput(outputFuture.get(1, TimeUnit.SECONDS))
        if (process.exitValue() != 0) {
            val detail = if (output.isBlank()) {
                "OSRM start command failed with exit code ${process.exitValue()}."
            } else {
                "OSRM start command failed with exit code ${process.exitValue()}. Output: $output"
            }
            throw ResponseStatusException(HttpStatus.INTERNAL_SERVER_ERROR, detail)
        }

        return unavailable.copy(
            status = "started",
            message = "OSRM start requested.",
            output = output,
        )
    }

    private fun resolveProjectDir(): File {
        RuntimeConfig.readConfigValue("OSRM_CONTROL_PROJECT_DIR")?.let { configured ->
            return File(configured).canonicalFile
        }
        var current = File(".").canonicalFile
        while (true) {
            if (File(current, DEFAULT_OSRM_CONTROL_COMPOSE_FILE).isFile) {
                return current
            }
            current = current.parentFile ?: return File(".").canonicalFile
        }
    }

    private fun resolveComposeFile(projectDir: File): File {
        val configured = RuntimeConfig.readConfigValue("OSRM_CONTROL_COMPOSE_FILE") ?: DEFAULT_OSRM_CONTROL_COMPOSE_FILE
        val file = File(configured)
        return if (file.isAbsolute) file.canonicalFile else File(projectDir, configured).canonicalFile
    }

    private fun resolveDockerBinary(): String? {
        RuntimeConfig.readConfigValue("OSRM_CONTROL_DOCKER_BIN")?.let { configured ->
            if (configured.contains(File.separator)) {
                return configured.takeIf { File(it).isFile }
            }
            return resolveExecutable(configured)
        }
        resolveExecutable("docker")?.let { return it }
        return listOf("/usr/local/bin/docker", "/opt/homebrew/bin/docker")
            .firstOrNull { candidate -> File(candidate).isFile }
    }

    private fun resolveExecutable(name: String): String? {
        return System.getenv("PATH")
            .orEmpty()
            .split(File.pathSeparator)
            .filter { it.isNotBlank() }
            .map { directory -> File(directory, name) }
            .firstOrNull { candidate -> candidate.isFile && candidate.canExecute() }
            ?.absolutePath
    }

    private fun osrmControlTimeoutMs(): Int {
        val configured = RuntimeConfig.readConfigValue("OSRM_CONTROL_TIMEOUT_MS")?.toIntOrNull()
        return (configured ?: DEFAULT_OSRM_CONTROL_TIMEOUT_MS).takeIf { it >= 1000 } ?: DEFAULT_OSRM_CONTROL_TIMEOUT_MS
    }

    private fun readBoolConfig(key: String, fallback: Boolean): Boolean {
        val normalized = RuntimeConfig.readConfigValue(key)?.lowercase(Locale.ROOT) ?: return fallback
        return when (normalized) {
            "1", "true", "yes", "y", "on" -> true
            "0", "false", "no", "n", "off" -> false
            else -> fallback
        }
    }

    private fun commandDisplay(parts: List<String>): String {
        return parts.joinToString(" ") { part ->
            if (part.any { it.isWhitespace() || it == '\'' || it == '"' }) {
                "'" + part.replace("'", "'\\''") + "'"
            } else {
                part
            }
        }
    }

    private fun trimOutput(output: String): String {
        val trimmed = output.trim()
        return if (trimmed.length <= MAX_OSRM_CONTROL_OUTPUT_CHARS) {
            trimmed
        } else {
            trimmed.takeLast(MAX_OSRM_CONTROL_OUTPUT_CHARS)
        }
    }
}
