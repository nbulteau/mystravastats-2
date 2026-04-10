import com.github.benmanes.gradle.versions.updates.DependencyUpdatesTask
import org.gradle.jvm.toolchain.JvmVendorSpec

plugins {
    kotlin("jvm") version "2.3.20"
    kotlin("plugin.spring") version "2.3.20"

    id("org.springframework.boot") version "4.0.5"
    id("io.spring.dependency-management") version "1.1.7"
    id("org.graalvm.buildtools.native") version "1.0.0"
    id("com.github.ben-manes.versions") version "0.53.0"
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

    implementation("tools.jackson.module:jackson-module-kotlin:3.1.1")

    implementation("org.springdoc:springdoc-openapi-starter-webmvc-ui:3.0.2")


    implementation("io.ktor:ktor-server-netty:3.4.2")

    implementation("com.squareup.okhttp3:okhttp:5.3.2")

    implementation("io.jenetics:jpx:4.0.0")

    implementation(files("libs/fit.jar"))

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
