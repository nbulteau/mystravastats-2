package me.nicolas.stravastats.api.configuration

import jakarta.servlet.http.HttpServletRequest
import jakarta.servlet.http.HttpServletResponse
import me.nicolas.stravastats.domain.RuntimeConfig
import org.springframework.http.MediaType
import org.springframework.stereotype.Component
import org.springframework.web.servlet.HandlerInterceptor
import tools.jackson.databind.json.JsonMapper

@Component
class MutationOriginInterceptor : HandlerInterceptor {
    private val safeMethods = setOf("GET", "HEAD", "OPTIONS")
    private val jsonMapper = JsonMapper.builder().build()

    override fun preHandle(
        request: HttpServletRequest,
        response: HttpServletResponse,
        handler: Any,
    ): Boolean {
        if (request.method in safeMethods) return true

        val origin = request.getHeader("Origin")?.trim().orEmpty()
        if (origin.isEmpty() || origin in RuntimeConfig.corsAllowedOrigins()) return true

        response.status = HttpServletResponse.SC_FORBIDDEN
        response.contentType = MediaType.APPLICATION_JSON_VALUE
        response.writer.write(jsonMapper.writeValueAsString(mapOf(
            "code" to 403,
            "message" to "cross-origin mutation rejected",
            "description" to "The request origin is not allowed to modify local data",
            "requestId" to response.getHeader("X-Request-Id"),
        )))
        return false
    }
}
