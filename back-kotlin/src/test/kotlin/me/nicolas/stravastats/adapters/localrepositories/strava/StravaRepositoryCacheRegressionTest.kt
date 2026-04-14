package me.nicolas.stravastats.adapters.localrepositories.strava

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assumptions.assumeTrue
import org.junit.jupiter.api.Test
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import kotlin.io.path.exists
import kotlin.io.path.isDirectory
import kotlin.io.path.name
import kotlin.io.path.pathString

class StravaRepositoryCacheRegressionTest {

    @Test
    fun `repository can deserialize real cache json samples when available`() {
        val cacheRoot = findRealCacheRoot()
        assumeTrue(cacheRoot != null, "real strava-cache directory not found")
        val resolvedCacheRoot = cacheRoot!!

        val activitiesFile = findFirstMatchingFile(resolvedCacheRoot, Regex("""activities-.+-\d{4}\.json"""))
        assumeTrue(activitiesFile != null, "no cached activities file found in strava-cache")

        val (clientId, year) = parseClientIdAndYear(activitiesFile!!.name)
        val tempCacheRoot = Files.createTempDirectory("strava-cache-regression")
        val targetYearDir = tempCacheRoot
            .resolve("strava-$clientId")
            .resolve("strava-$clientId-$year")
        Files.createDirectories(targetYearDir)
        Files.copy(activitiesFile, targetYearDir.resolve(activitiesFile.name), StandardCopyOption.REPLACE_EXISTING)

        val repository = StravaRepository(tempCacheRoot.pathString)
        val activities = repository.loadActivitiesFromCache(clientId, year)

        val detailedFile = findFirstMatchingFile(activitiesFile.parent, Regex("""stravaActivity-\d+"""))
        if (detailedFile != null) {
            Files.copy(detailedFile, targetYearDir.resolve(detailedFile.name), StandardCopyOption.REPLACE_EXISTING)
            val detailedId = detailedFile.name.removePrefix("stravaActivity-").toLong()
            val detailed = repository.loadDetailedActivityFromCache(clientId, year, detailedId)
            assertNotNull(detailed, "expected detailed activity to be deserialized")
            assertEquals(detailedId, detailed!!.id)
        }

        val streamFile = findFirstMatchingFile(activitiesFile.parent, Regex("""stream-\d+"""))
        if (streamFile != null) {
            Files.copy(streamFile, targetYearDir.resolve(streamFile.name), StandardCopyOption.REPLACE_EXISTING)
            val streamActivityId = streamFile.name.removePrefix("stream-").toLong()
            val activityForStream: StravaActivity = activities.firstOrNull { it.id == streamActivityId }
                ?: activities.first().copy(id = streamActivityId)
            val stream = repository.loadActivitiesStreamsFromCache(clientId, year, activityForStream)
            assertNotNull(stream, "expected stream to be deserialized")
        }
    }

    private fun parseClientIdAndYear(activitiesFileName: String): Pair<String, Int> {
        val regex = Regex("""activities-(.+)-(\d{4})\.json""")
        val match = regex.matchEntire(activitiesFileName)
            ?: throw IllegalArgumentException("invalid activities filename: $activitiesFileName")
        return match.groupValues[1] to match.groupValues[2].toInt()
    }

    private fun findFirstMatchingFile(root: Path, regex: Regex): Path? {
        Files.walk(root).use { paths ->
            return paths
                .filter { Files.isRegularFile(it) }
                .filter { regex.matches(it.name) }
                .findFirst()
                .orElse(null)
        }
    }

    private fun findRealCacheRoot(): Path? {
        val candidates = listOf(
            Path.of("../strava-cache"),
            Path.of("../../strava-cache"),
            Path.of("strava-cache"),
        )
        return candidates.firstOrNull { it.exists() && it.isDirectory() }
    }
}
