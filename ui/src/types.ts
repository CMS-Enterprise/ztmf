/**
 * Type definitions for the application.
 * @module types
 */
import type {
  CognitoIdToken,
  CognitoUser,
  CognitoUserSession,
} from 'amazon-cognito-identity-js'

export type AppConfig = {
  AWS_REGION: string | 'us-east-1'
  API_URL: string
  CF_DOMAIN: string
  COGNITO_DOMAIN: string
  COGNITO_REDIRECT_SIGN_IN: string
  COGNITO_REDIRECT_SIGN_OUT: string
  USER_POOL_CLIENT_ID: string
  USER_POOL_ID: string
} & AppFeatureFlags

export type AppFeatureFlags = {
  IDP_ENABLED: boolean
}

export type CognitoUserInfo = {
  attributes: {
    email: string
    sub?: string
  }
  id?: string
  username: string
}

export type UserState = {
  user: CognitoUser
  userInfo: CognitoUserInfo
  userSession: CognitoUserSession
  idToken: CognitoIdToken
  jwtToken: string
}

export type UserDataType =
  | (UserState &
      CognitoUserInfo & {
        role?: string
        avatar?: string | null
        email?: string
        fullName?: string
        password?: string
      })
  | null

export type FormField = {
  name: string
  label?: string
  type: string
  required?: boolean
  disabled?: boolean
  multiline?: boolean
  value?: string | number | boolean | null
  component: React.ElementType
}

export type ThemeColor =
  | 'primary'
  | 'secondary'
  | 'error'
  | 'warning'
  | 'info'
  | 'success'

export type ThemeSkin = 'filled' | 'light' | 'light-static'
