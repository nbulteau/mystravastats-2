package me.nicolas.stravastats.domain.services.activityproviders
import org.slf4j.LoggerFactory
import java.time.Instant
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicLong
/**
 * Manages Strava API rate-limit detection and cooldown.
 * After a HTTP 429, enters a [RATE_LIMIT_COOLDOWN_MS]-ms cooldown where no API calls are made.
 */
internal class StravaRateLimiter {
    private val logger = LoggerFactory.getLogger(StravaRateLimiter::class.java)
    /** Epoch ms until which API calls must not be sent (0 = no active limit). */
    private val rateLimitUntilMs = AtomicLong(0L)
    companion object {
        /** Strava resets its 15-minute window every 15 minutes, so a full window is used. */
        val RATE_LIMIT_COOLDOWN_MS: Long = TimeUnit.MINUTES.toMillis(15)
    }
    /** Returns true if the rate-limit cooldown is currently active. */
    fun isActive(): Boolean = rateLimitUntilMs.get() > System.currentTimeMillis()
    /** Returns the epoch-ms timestamp until which the rate limit is active (0 if inactive). */
    fun untilEpochMs(): Long = rateLimitUntilMs.get()
    /**
     * Records a rate-limit event and activates the cooldown window.
     * Idempotent: only the first call per window updates state and logs a warning.
     */
    fun mark(source: String, throwable: Throwable? = null) {
        val now = System.currentTimeMillis()
        val until = now + RATE_LIMIT_COOLDOWN_MS
        val previous = rateLimitUntilMs.getAndUpdate { current -> maxOf(current, until) }
        if (previous > now) {
            return
        }
        logger.warn(
            "Strava rate limit detected ({}). Switching to cache-only mode until {}",
            source,
            Instant.ofEpochMilli(until),
        )
        if (throwable != null) {
            logger.debug("Rate limit trigger details for '{}'", source, throwable)
        }
    }
}
