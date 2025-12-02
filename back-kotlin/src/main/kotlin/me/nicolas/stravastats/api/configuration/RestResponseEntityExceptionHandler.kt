package me.nicolas.stravastats.api.configuration

import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import org.springframework.data.rest.webmvc.ResourceNotFoundException
import org.springframework.http.HttpHeaders
import org.springframework.http.HttpStatus
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.ControllerAdvice
import org.springframework.web.bind.annotation.ExceptionHandler
import org.springframework.web.context.request.WebRequest
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException
import org.springframework.web.servlet.mvc.method.annotation.ResponseEntityExceptionHandler


@ControllerAdvice
 class RestResponseEntityExceptionHandler

    : ResponseEntityExceptionHandler() {
    @ExceptionHandler(value = [ResourceNotFoundException::class])
    protected fun handleResourceNotFound(
        exception: ResourceNotFoundException,
        request: WebRequest?,
    ): ResponseEntity<Any>? {

        val errorResponse = ErrorResponseMessageDto(
            message = "Resource not found",
            description = "Illegal argument : '${exception.message}'",
            code = 1
        )

        return handleExceptionInternal(exception, errorResponse, HttpHeaders(), HttpStatus.NOT_FOUND, request!!)
    }

    @ExceptionHandler(value = [IllegalArgumentException::class, IllegalStateException::class])
    protected fun handleConflict(exception: RuntimeException, request: WebRequest?): ResponseEntity<Any>? {

        val errorResponse = ErrorResponseMessageDto(
            message = "Illegal argument",
            description = "Illegal argument : '${exception.message}'",
            code = 1
        )

        return handleExceptionInternal(exception, errorResponse, HttpHeaders(), HttpStatus.BAD_REQUEST, request!!)
    }

    @ExceptionHandler(value = [MethodArgumentTypeMismatchException::class])
    protected fun handleMethodArgumentTypeMismatchException(
        exception: MethodArgumentTypeMismatchException,
        request: WebRequest?,
    ): ResponseEntity<Any>? {
        val bodyOfResponse = "Unknown argument"

        when (exception.name) {
            "activityType" -> {
                val errorResponse = ErrorResponseMessageDto(
                    message = "Unknown stravaActivity type",
                    description = "The stravaActivity type '${exception.value}' is not recognized.",
                    code = 1
                )
                return handleExceptionInternal(
                    exception,
                    errorResponse,
                    HttpHeaders(),
                    HttpStatus.BAD_REQUEST,
                    request!!
                )
            }

            "year" -> {
                val errorResponse = ErrorResponseMessageDto(
                    message = "Invalid year value",
                    description = "The year '${exception.value}' is not valid.",
                    code = 1
                )
                return handleExceptionInternal(
                    exception,
                    errorResponse,
                    HttpHeaders(),
                    HttpStatus.BAD_REQUEST,
                    request!!
                )
            }

            "period" -> {
                val errorResponse = ErrorResponseMessageDto(
                    message = "Invalid period value",
                    description = "The period '${exception.value}' is not valid.",
                    code = 1
                )
                return handleExceptionInternal(
                    exception,
                    errorResponse,
                    HttpHeaders(),
                    HttpStatus.BAD_REQUEST,
                    request!!
                )
            }
        }

        return handleExceptionInternal(exception, bodyOfResponse, HttpHeaders(), HttpStatus.CONFLICT, request!!)
    }
}