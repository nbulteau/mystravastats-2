package me.nicolas.stravastats.api.configuration

import org.springframework.context.annotation.Configuration
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.method.HandlerTypePredicate
import org.springframework.web.servlet.config.annotation.PathMatchConfigurer
import org.springframework.web.servlet.config.annotation.InterceptorRegistry
import org.springframework.web.servlet.config.annotation.ViewControllerRegistry
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer

@Configuration
class WebConfig(
    private val mutationOriginInterceptor: MutationOriginInterceptor,
) : WebMvcConfigurer {

    override fun addInterceptors(registry: InterceptorRegistry) {
        registry.addInterceptor(mutationOriginInterceptor).addPathPatterns("/api/**")
    }

    override fun configurePathMatch(configurer: PathMatchConfigurer) {
        // Keep controller mappings clean while exposing API under /api.
        configurer.addPathPrefix("/api", HandlerTypePredicate.forAnnotation(RestController::class.java))
    }

    override fun addViewControllers(registry: ViewControllerRegistry) {
        // Explicit Vue routes keep unknown API paths out of the SPA fallback.
        listOf(
            "/", "/statistics", "/gear", "/activities", "/map", "/charts", "/dashboard",
            "/annual-recap", "/commute-recap", "/heatmap", "/segments", "/routes", "/diagnostics",
            "/settings", "/badges", "/badges/climbs/{variantId}", "/activities/{id}",
        ).forEach { registry.addViewController(it).setViewName("forward:/index.html") }
    }
}
