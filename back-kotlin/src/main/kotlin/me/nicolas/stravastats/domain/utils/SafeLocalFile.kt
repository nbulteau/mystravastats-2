package me.nicolas.stravastats.domain.utils

import java.io.File
import java.nio.ByteBuffer
import java.nio.channels.FileChannel
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import java.nio.file.StandardOpenOption
import java.nio.file.attribute.PosixFilePermission
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.locks.ReentrantLock
import kotlin.concurrent.withLock

object SafeLocalFile {
    private val locks = ConcurrentHashMap<String, ReentrantLock>()
    private val ownerOnlyFilePermissions = setOf(
        PosixFilePermission.OWNER_READ,
        PosixFilePermission.OWNER_WRITE,
    )
    private val ownerOnlyDirectoryPermissions = ownerOnlyFilePermissions + PosixFilePermission.OWNER_EXECUTE

    fun <T> withLock(file: File, operation: () -> T): T {
        val key = file.toPath().toAbsolutePath().normalize().toString()
        return locks.computeIfAbsent(key) { ReentrantLock() }.withLock(operation)
    }

    fun write(file: File, content: ByteArray) {
        val path = file.toPath().toAbsolutePath().normalize()
        val directory = path.parent ?: error("Local file must have a parent directory: $path")
        Files.createDirectories(directory)
        setPermissionsIfSupported(directory, ownerOnlyDirectoryPermissions)

        if (Files.exists(path)) {
            writeAtomically(path.resolveSibling("${path.fileName}.bak"), Files.readAllBytes(path))
        }
        writeAtomically(path, content)
    }

    private fun writeAtomically(path: java.nio.file.Path, content: ByteArray) {
        val temporary = Files.createTempFile(path.parent, ".${path.fileName}.tmp-", "")
        try {
            setPermissionsIfSupported(temporary, ownerOnlyFilePermissions)
            FileChannel.open(temporary, StandardOpenOption.WRITE, StandardOpenOption.TRUNCATE_EXISTING).use { channel ->
                var buffer = ByteBuffer.wrap(content)
                while (buffer.hasRemaining()) {
                    channel.write(buffer)
                }
                channel.force(true)
            }
            Files.move(
                temporary,
                path,
                StandardCopyOption.ATOMIC_MOVE,
                StandardCopyOption.REPLACE_EXISTING,
            )
        } finally {
            Files.deleteIfExists(temporary)
        }
    }

    private fun setPermissionsIfSupported(path: java.nio.file.Path, permissions: Set<PosixFilePermission>) {
        try {
            Files.setPosixFilePermissions(path, permissions)
        } catch (_: UnsupportedOperationException) {
            // POSIX permissions are unavailable on this filesystem.
        }
    }
}
