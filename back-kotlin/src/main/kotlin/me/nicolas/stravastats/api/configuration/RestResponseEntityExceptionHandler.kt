package me.nicolas.stravastats.api.configuration

import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import org.springframework.data.rest.webmvc.ResourceNotFoundException
import org.springframework.http.HttpHeaders
import org.springframework.http.HttpStatus
import org.springframework.http.HttpStatusCode
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.MissingServletRequestParameterException
import org.springframework.web.bind.annotation.ControllerAdvice
import org.springframework.web.bind.annotation.ExceptionHandler
import org.springframework.web.context.request.WebRequest
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException
import org.springframework.web.servlet.mvc.method.annotation.ResponseEntityExceptionHandler

@ControllerAdvice
class RestResponseEntityExceptionHandler : ResponseEntityExceptionHandler() {

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

    override fun handleMissingServletRequestParameter(
        exception: MissingServletRequestParameterException,
        headers: HttpHeaders,
        status: HttpStatusCode,
        request: WebRequest,
    ): ResponseEntity<Any>? {
        val errorResponse = ErrorResponseMessageDto(
            message = "Missing required request parameter",
            description = "The parameter '${exception.parameterName}' is required.",
            code = 1
        )

        return handleExceptionInternal(
            exception,
            errorResponse,
            headers,
            status,
            request
        )
    }

    @ExceptionHandler(value = [MethodArgumentTypeMismatchException::class])
    protected fun handleMethodArgumentTypeMismatchException(
        exception: MethodArgumentTypeMismatchException,
        request: WebRequest?,
    ): ResponseEntity<Any>? {
        val errorResponse = when (exception.name) {
            "activityType" -> ErrorResponseMessageDto(
                message = "Unknown stravaActivity type",
                description = "The stravaActivity type '${exception.value}' is not recognized.",
                code = 1
            )

            "year" -> ErrorResponseMessageDto(
                message = "Invalid year value",
                description = "The year '${exception.value}' is not valid.",
                code = 1
            )

            "period" -> ErrorResponseMessageDto(
                message = "Invalid period value",
                description = "The period '${exception.value}' is not valid.",
                code = 1
            )

            "badgeSet" -> ErrorResponseMessageDto(
                message = "Invalid badgeSet value",
                description = "The badgeSet '${exception.value}' is not valid.",
                code = 1
            )

            "activityId" -> ErrorResponseMessageDto(
                message = "Invalid activityId value",
                description = "The activityId '${exception.value}' is not valid.",
                code = 1
            )

            else -> ErrorResponseMessageDto(
                message = "Invalid request parameter value",
                description = "The parameter '${exception.name}' has invalid value '${exception.value}'.",
                code = 1
            )
        }

        return handleExceptionInternal(
            exception,
            errorResponse,
            HttpHeaders(),
            HttpStatus.BAD_REQUEST,
            request!!
        )
    }
}
