package me.nicolas.stravastats.api.configuration

import jakarta.servlet.FilterChain
import jakarta.servlet.http.HttpServletRequest
import jakarta.servlet.http.HttpServletRequestWrapper
import jakarta.servlet.http.HttpServletResponse
import org.springframework.core.Ordered
import org.springframework.core.annotation.Order
import org.springframework.stereotype.Component
import org.springframework.web.filter.OncePerRequestFilter
import java.util.Collections
import java.util.UUID

@Component
@Order(Ordered.HIGHEST_PRECEDENCE)
class ApiRequestContextFilter : OncePerRequestFilter() {
    override fun shouldNotFilter(request: HttpServletRequest): Boolean {
        val path = request.requestURI.removePrefix(request.contextPath)
        return path != "/api" && !path.startsWith("/api/")
    }

    override fun doFilterInternal(request: HttpServletRequest, response: HttpServletResponse, filterChain: FilterChain) {
        val candidate = request.getHeader("X-Request-Id")?.trim().orEmpty()
        val requestId = candidate.takeIf { it.matches(Regex("[A-Za-z0-9._:-]{1,128}")) }
            ?: UUID.randomUUID().toString()
        response.setHeader("X-Request-Id", requestId)
        val wrapped = object : HttpServletRequestWrapper(request) {
            override fun getHeader(name: String): String? =
                if (name.equals("X-Request-Id", ignoreCase = true)) requestId else super.getHeader(name)

            override fun getHeaders(name: String): java.util.Enumeration<String> =
                if (name.equals("X-Request-Id", ignoreCase = true)) Collections.enumeration(listOf(requestId))
                else super.getHeaders(name)

            override fun getHeaderNames(): java.util.Enumeration<String> =
                Collections.enumeration(super.getHeaderNames().toList().filterNot {
                    it.equals("X-Request-Id", ignoreCase = true)
                } + "X-Request-Id")
        }
        filterChain.doFilter(wrapped, response)
    }
}
