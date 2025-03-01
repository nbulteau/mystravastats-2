plugins {
    kotlin("jvm") version "2.1.0"
    kotlin("plugin.spring") version "2.1.0"
    id("org.springframework.boot") version "3.4.3"
    id("io.spring.dependency-management") version "1.1.7"
    id("com.github.ben-manes.versions") version "0.52.0"
    id("org.graalvm.buildtools.native") version "0.10.5"
}

group = "me.nicolas"
version = "0.0.1-SNAPSHOT"

repositories {
    mavenCentral()
    maven("https://repo.kotlin.link")
}

dependencies {
    implementation("org.jetbrains.kotlin:kotlin-reflect")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-core")

    implementation("org.springframework.boot:spring-boot-starter-actuator")
    implementation("org.springframework.boot:spring-boot-starter-web")
    implementation("org.springframework.boot:spring-boot-starter-data-rest")

    //developmentOnly("org.springframework.boot:spring-boot-devtools")
    //providedRuntime("org.springframework.boot:spring-boot-starter-tomcat")

    implementation("com.fasterxml.jackson.module:jackson-module-kotlin")

    implementation("org.springdoc:springdoc-openapi-starter-webmvc-ui:2.8.5")

    implementation("io.ktor:ktor-server-netty:3.1.1")

    implementation("com.squareup.okhttp3:okhttp:4.12.0")

    implementation("io.jenetics:jpx:3.2.1")

    implementation(files("libs/fit.jar"))

    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
    testImplementation("org.springframework.boot:spring-boot-starter-test") {
        exclude(module = "mockito-core")
    }
    testImplementation("org.jetbrains.kotlin:kotlin-test-junit5")
    testImplementation("com.ninja-squad:springmockk:4.0.2")
}

kotlin {
    compilerOptions {
        freeCompilerArgs.addAll("-Xjsr305=strict")
    }
}

tasks.withType<Test> {
    useJUnitPlatform()
}



graalvmNative {
    binaries {
        named("main") {
            imageName.set("mystravastats")
            mainClass.set("me.nicolas.stravastats.StravastatsApplicationKt")
            debug.set(true)
            verbose.set(true)
            fallback.set(true)
            sharedLibrary.set(false)
            richOutput.set(false)
            quickBuild.set(false)

            // Advanced options
            buildArgs.add("--link-at-build-time")

            jvmArgs.add("-Xmx2g")

            // Runtime options
            runtimeArgs.add("--help")

            useFatJar.set(true)
        }
    }
}
