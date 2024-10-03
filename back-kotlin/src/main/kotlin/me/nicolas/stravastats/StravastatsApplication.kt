package me.nicolas.stravastats

import org.springframework.boot.autoconfigure.SpringBootApplication
import org.springframework.boot.runApplication

@SpringBootApplication
class StravastatsApplication

fun main(args: Array<String>) {
    runApplication<StravastatsApplication>(*args)
}
