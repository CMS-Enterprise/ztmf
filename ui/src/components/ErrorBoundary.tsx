/**
 * Generic React error boundary component.
 * @module components/ErrorBoundary
 */
import { useNavigate, useRouteError } from 'react-router-dom'
import { ErrorResponse } from '@remix-run/router'
import Container from '@mui/material/Container'
import { Routes } from '@/router/constants'
import {
  GENERIC_ERROR_MESSAGE,
  NOT_FOUND_ERROR_MESSAGE,
  SESSION_EXPIRED_MESSAGE,
  UNAUTHORIZED_ACCESS_MESSAGE,
} from '@/locales/en'
import Typography from '@mui/material/Typography'

const ErrorMessageMap = new Map<number, string>([
  [400, GENERIC_ERROR_MESSAGE],
  [401, SESSION_EXPIRED_MESSAGE],
  [403, UNAUTHORIZED_ACCESS_MESSAGE],
  [404, NOT_FOUND_ERROR_MESSAGE],
])

const useErrorMessage = (error: ErrorResponse): string =>
  ErrorMessageMap.get((error.status || 400) ?? 0) ?? GENERIC_ERROR_MESSAGE

const useErrorTitle = (error: ErrorResponse): string => {
  if (error.status && error.statusText) {
    return `${error.status} ${error.statusText}`
  }

  return `Uknown Error`
}

/**
 * A generic React error boundary component.
 * @returns {JSX.Element}
 */
const ErrorBoundary: React.FC = (): JSX.Element => {
  const navigate = useNavigate()
  const error = useRouteError() as ErrorResponse
  const title = useErrorTitle(error)
  const message = useErrorMessage(error)

  if (error.status === 401) {
    setTimeout(() => navigate(Routes.AUTH_LOGOUT), 5000)
  }

  console.log({ error })

  return (
    <Container sx={{ m: 3 }}>
      <Typography variant="h2">{title}</Typography>
      {error.data && <Typography variant="h3">{error.data}</Typography>}
      <p aria-label="error-message">
        <strong>{message}</strong>
        <pre>{JSON.stringify(error)}</pre>
      </p>
    </Container>
  )
}

export default ErrorBoundary
