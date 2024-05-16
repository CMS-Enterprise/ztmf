/**
 * Set global configuration for the application provided by Vite environment variables
 * @module utils/config
 * @exports CONFIG
 */
import type { AppConfig } from '@/types'

const CONFIG = {
  //* AWS configuration
  AWS_REGION: `${process.env.VITE_AWS_REGION}`,
  CF_DOMAIN: `${process.env.VITE_CF_DOMAIN}`,
  API_URL: `${process.env.VITE_CF_DOMAIN}/api`,
  COGNITO_DOMAIN: `${process.env.VITE_COGNITO_DOMAIN}`,
  COGNITO_REDIRECT_SIGN_IN: `${process.env.VITE_COGNITO_REDIRECT_SIGN_IN}`,
  COGNITO_REDIRECT_SIGN_OUT: `${process.env.VITE_COGNITO_REDIRECT_SIGN_OUT}`,
  USER_POOL_CLIENT_ID: `${process.env.VITE_USER_POOL_CLIENT_ID}`,
  USER_POOL_ID: `${process.env.VITE_USER_POOL_ID}`,

  //* Feature flags
  IDP_ENABLED: Boolean(process.env.VITE_IDP_ENABLED),
} satisfies AppConfig

export default CONFIG
