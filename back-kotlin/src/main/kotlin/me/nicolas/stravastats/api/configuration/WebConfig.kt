package me.nicolas.stravastats.api.configuration

import org.springframework.context.annotation.Configuration
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.method.HandlerTypePredicate
import org.springframework.web.servlet.config.annotation.PathMatchConfigurer
import org.springframework.web.servlet.config.annotation.ViewControllerRegistry
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer

@Configuration
class WebConfig : WebMvcConfigurer {

    override fun configurePathMatch(configurer: PathMatchConfigurer) {
        // Keep controller mappings clean while exposing API under /api.
        configurer.addPathPrefix("/api", HandlerTypePredicate.forAnnotation(RestController::class.java))
    }

    override fun addViewControllers(registry: ViewControllerRegistry) {
        // Forward client-side routes to the Vue entry point.
        registry.addViewController("/{spring:[^.]*}").setViewName("forward:/index.html")
        registry.addViewController("/**/{spring:[^.]*}").setViewName("forward:/index.html")
    }
}
