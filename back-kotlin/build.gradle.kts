import com.github.benmanes.gradle.versions.updates.DependencyUpdatesTask
import org.gradle.api.GradleException
import org.gradle.api.tasks.Exec
import org.gradle.api.tasks.Sync
import org.gradle.jvm.toolchain.JvmVendorSpec
import org.gradle.language.jvm.tasks.ProcessResources

plugins {
    kotlin("jvm") version "2.4.0"
    kotlin("plugin.spring") version "2.4.0"

    id("org.springframework.boot") version "4.0.6"
    id("io.spring.dependency-management") version "1.1.7"
    id("org.graalvm.buildtools.native") version "1.0.0"
    id("com.github.ben-manes.versions") version "0.54.0"
}

group = "me.nicolas"
version = "0.0.1-SNAPSHOT"

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(25)
    }
}

springBoot {
    mainClass.set("me.nicolas.stravastats.StravastatsApplicationKt")
}

repositories {
    mavenCentral()
    maven("https://repo.kotlin.link")
    // JitPack is used to resolve the official Garmin FIT Java SDK from GitHub.
    maven("https://jitpack.io")
}

dependencies {
    implementation("org.jetbrains.kotlin:kotlin-reflect")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core")

    implementation("org.springframework.boot:spring-boot-starter-actuator")
    implementation("org.springframework.boot:spring-boot-starter-webmvc")
    implementation("org.springframework.boot:spring-boot-starter-data-rest")
    implementation("org.springframework.boot:spring-boot-starter-hateoas")

    //developmentOnly("org.springframework.boot:spring-boot-devtools")
    //providedRuntime("org.springframework.boot:spring-boot-starter-tomcat")

    implementation("tools.jackson.module:jackson-module-kotlin:3.1.4")

    implementation("org.springdoc:springdoc-openapi-starter-webmvc-ui:3.0.3")

    implementation("com.squareup.okhttp3:okhttp:5.3.2")

    implementation("io.jenetics:jpx:4.0.0")

    // Garmin FIT Java SDK sourced from GitHub via JitPack (replaces the vendored libs/fit.jar).
    // All SessionMesg/RecordMesg accessors used by FITRepository are present and compatible.
    // Newer versions available at https://github.com/garmin/fit-java-sdk/tags.
    implementation("com.github.garmin:fit-java-sdk:21.202.0")

    testRuntimeOnly("org.junit.platform:junit-platform-launcher")

    testImplementation("org.springframework.boot:spring-boot-starter-webmvc-test")
    testImplementation("org.jetbrains.kotlin:kotlin-test-junit5")
    testImplementation("com.ninja-squad:springmockk:5.0.1")
}

kotlin {
    compilerOptions {
        freeCompilerArgs.addAll("-Xjsr305=strict")
    }
}

tasks.withType<Test> {
    useJUnitPlatform()
}

val npmExecutable = if (System.getProperty("os.name").lowercase().contains("windows")) "npm.cmd" else "npm"
val frontendProjectDir = layout.projectDirectory.dir("../front-vue")
val frontendDistDir = frontendProjectDir.dir("dist")
val generatedFrontendStaticDir = layout.buildDirectory.dir("generated/frontend-static")
val includeFrontendAssetsRequested = providers.gradleProperty("includeFrontendAssets")
    .map { it.equals("true", ignoreCase = true) || it == "1" }
    .getOrElse(
        gradle.startParameter.taskNames.any { taskName ->
            taskName == "bootJarWithFrontend" || taskName.endsWith(":bootJarWithFrontend")
        }
    )

val installFrontendDependencies = tasks.register<Exec>("installFrontendDependencies") {
    group = "build"
    description = "Installs frontend dependencies from the lockfile before a standalone backend package."
    workingDir = frontendProjectDir.asFile
    commandLine(npmExecutable, "ci", "--loglevel=error", "--no-audit", "--no-fund", "--update-notifier=false")
    doFirst {
        if (!frontendProjectDir.file("package.json").asFile.isFile) {
            throw GradleException("Cannot build frontend assets: ../front-vue/package.json is missing.")
        }
    }
}

val buildFrontendAssets = tasks.register<Exec>("buildFrontendAssets") {
    group = "build"
    description = "Builds the Vue frontend production bundle."
    dependsOn(installFrontendDependencies)
    workingDir = frontendProjectDir.asFile
    environment("VITE_CJS_TRACE", "false")
    environment("NODE_OPTIONS", "--no-deprecation")
    commandLine(npmExecutable, "run", "build")
}

val syncFrontendAssets = tasks.register<Sync>("syncFrontendAssets") {
    group = "build"
    description = "Copies the fresh Vue bundle into the generated Spring static resources directory."
    dependsOn(buildFrontendAssets)
    from(frontendDistDir)
    into(generatedFrontendStaticDir)
    doFirst {
        if (!frontendDistDir.file("index.html").asFile.isFile) {
            throw GradleException("Cannot sync frontend assets: ../front-vue/dist/index.html is missing.")
        }
    }
}

if (includeFrontendAssetsRequested) {
    tasks.named<ProcessResources>("processResources") {
        dependsOn(syncFrontendAssets)
        from(generatedFrontendStaticDir) {
            into("static")
        }
    }
}

tasks.register("bootJarWithFrontend") {
    group = "build"
    description = "Builds the Spring Boot jar with freshly generated Vue static assets."
    dependsOn(tasks.named("bootJar"))
}

// Keep plain jar enabled: Spring AOT/native compile relies on the application artifact/classpath.
tasks.named<Jar>("jar") {
    enabled = true
}

// https://github.com/ben-manes/gradle-versions-plugin
fun isNonStable(version: String): Boolean {
    val stableKeyword = listOf("RELEASE", "FINAL", "GA").any { version.uppercase().contains(it) }
    val regex = "^[0-9,.v-]+(-r)?$".toRegex()
    val isStable = stableKeyword || regex.matches(version)
    return isStable.not()
}

tasks.withType<DependencyUpdatesTask> {
    rejectVersionIf {
        isNonStable(candidate.version) && !isNonStable(currentVersion)
    }
}

graalvmNative {
    toolchainDetection.set(true)
    binaries {
        named("main") {
            imageName.set("mystravastats-kotlin")
            // Keep a strict native binary without JVM fallback.
            buildArgs.add("--no-fallback")
            javaLauncher.set(
                javaToolchains.launcherFor {
                    languageVersion = JavaLanguageVersion.of(25)
                    vendor = JvmVendorSpec.GRAAL_VM
                }
            )
        }
    }
}
