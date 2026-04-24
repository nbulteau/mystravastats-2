package me.nicolas.stravastats.domain.utils

import me.nicolas.stravastats.domain.RuntimeConfig
import java.awt.Desktop
import java.net.URI

object BrowserUtils {
    fun openBrowser(url: String) {
        if (isBrowserAutoOpenDisabled()) {
            println("Browser auto-open disabled; open this URL manually:")
            println(url)
            return
        }

        if (isNativeImage()) {
            println("To grant MyStravaStats to read your Strava activities data: copy paste this URL in a browser")
        } else {
            try {
                if (Desktop.isDesktopSupported()) {
                    Desktop.getDesktop().browse(URI(url))
                } else {
                    openBrowserWithRuntimeExec(url)
                }
            } catch (exception: Exception) {
                println("Unable to open browser: ${exception.message}")
            }
        }
    }

    // Fallback for environments where Desktop is not supported
    private fun openBrowserWithRuntimeExec(url: String) {
        val os = System.getProperty("os.name").lowercase()
        val command = when {
            // Important: fournir un titre vide ("") et mettre l'URL entre guillemets
            os.contains("win") -> arrayOf("cmd", "/c", "start", "\"\"", "\"$url\"")
            os.contains("mac") -> arrayOf("open", url)
            else -> arrayOf("xdg-open", url)  // Linux et autres
        }
        Runtime.getRuntime().exec(command)
    }

    private fun isNativeImage(): Boolean {
        return System.getProperty("org.graalvm.nativeimage.imagecode") != null
    }

    private fun isBrowserAutoOpenDisabled(): Boolean {
        return when (RuntimeConfig.readConfigValue("OPEN_BROWSER")?.lowercase()) {
            "0", "false", "no", "off" -> true
            else -> false
        }
    }
}
