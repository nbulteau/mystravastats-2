package me.nicolas.stravastats.api.configuration

import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.springframework.http.HttpHeaders
import org.springframework.mock.web.MockFilterChain
import org.springframework.mock.web.MockHttpServletRequest
import org.springframework.mock.web.MockHttpServletResponse

class CorsConfigTest {
    @AfterEach
    fun tearDown() {
        System.clearProperty("CORS_ALLOWED_ORIGINS")
    }

    @Test
    fun `preflight allows configured origin with credentials`() {
        System.setProperty("CORS_ALLOWED_ORIGINS", "https://app.example")

        val response = performPreflight(
            origin = "https://app.example",
            requestMethod = "GET",
            requestHeaders = "authorization,x-request-id",
        )

        assertEquals(200, response.status)
        assertEquals("https://app.example", response.getHeader(HttpHeaders.ACCESS_CONTROL_ALLOW_ORIGIN))
        assertEquals("true", response.getHeader(HttpHeaders.ACCESS_CONTROL_ALLOW_CREDENTIALS))
        assertTrue(headerContains(response.getHeader(HttpHeaders.ACCESS_CONTROL_ALLOW_METHODS), "GET"))
        assertTrue(headerContains(response.getHeader(HttpHeaders.ACCESS_CONTROL_ALLOW_HEADERS), "Authorization"))
        assertTrue(headerContains(response.getHeader(HttpHeaders.ACCESS_CONTROL_ALLOW_HEADERS), "X-Request-Id"))
    }

    @Test
    fun `preflight rejects unconfigured origin`() {
        System.setProperty("CORS_ALLOWED_ORIGINS", "https://app.example")

        val response = performPreflight(
            origin = "https://evil.example",
            requestMethod = "GET",
            requestHeaders = "Authorization",
        )

        assertEquals(403, response.status)
        assertNull(response.getHeader(HttpHeaders.ACCESS_CONTROL_ALLOW_ORIGIN))
        assertNull(response.getHeader(HttpHeaders.ACCESS_CONTROL_ALLOW_CREDENTIALS))
    }

    private fun performPreflight(
        origin: String,
        requestMethod: String,
        requestHeaders: String,
    ): MockHttpServletResponse {
        val request = MockHttpServletRequest("OPTIONS", "/api/health/details")
        request.addHeader(HttpHeaders.ORIGIN, origin)
        request.addHeader(HttpHeaders.ACCESS_CONTROL_REQUEST_METHOD, requestMethod)
        request.addHeader(HttpHeaders.ACCESS_CONTROL_REQUEST_HEADERS, requestHeaders)

        val response = MockHttpServletResponse()
        CorsConfig().corsFilter().doFilter(request, response, MockFilterChain())
        return response
    }

    private fun headerContains(header: String?, expected: String): Boolean {
        return header
            ?.split(',')
            ?.any { part -> part.trim().equals(expected, ignoreCase = true) }
            ?: false
    }
}
