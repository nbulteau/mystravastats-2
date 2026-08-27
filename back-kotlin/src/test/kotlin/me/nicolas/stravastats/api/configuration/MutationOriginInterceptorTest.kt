package me.nicolas.stravastats.api.configuration

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.springframework.mock.web.MockHttpServletRequest
import org.springframework.mock.web.MockHttpServletResponse

class MutationOriginInterceptorTest {
    private val interceptor = MutationOriginInterceptor()

    @Test
    fun `allows safe cross-origin reads`() {
        val request = MockHttpServletRequest("GET", "/api/health/details")
        request.addHeader("Origin", "https://example.test")

        assertTrue(interceptor.preHandle(request, MockHttpServletResponse(), Any()))
    }

    @Test
    fun `allows local and command-line mutations`() {
        val localRequest = MockHttpServletRequest("POST", "/api/source-modes/apply")
        localRequest.addHeader("Origin", "http://localhost:5173")
        val cliRequest = MockHttpServletRequest("PUT", "/api/athletes/me/performance-settings")

        assertTrue(interceptor.preHandle(localRequest, MockHttpServletResponse(), Any()))
        assertTrue(interceptor.preHandle(cliRequest, MockHttpServletResponse(), Any()))
    }

    @Test
    fun `rejects browser mutations from an untrusted origin`() {
        val request = MockHttpServletRequest("DELETE", "/api/data-quality/corrections/123")
        request.addHeader("Origin", "https://example.test")
        val response = MockHttpServletResponse()

        assertFalse(interceptor.preHandle(request, response, Any()))
        assertEquals(403, response.status)
        assertEquals("application/json", response.contentType)
    }
}
