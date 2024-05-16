/**
 * Types used in custom hooks.
 * @module hooks/types
 */

/**
 * @typedef {Object} LoginParams - Parameters for the login hook.
 * @property {string} email - The user's email address.
 * @property {string} password - The user's password.
 * @see {@link dashboard/actions/loginUser}
 */
export type LoginParams = {
  email: string
  password: string
}
