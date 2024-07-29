/**
 * Auth state loader for react-router data routes.
 * @module router/authLoader
 * @see {@link dashboard/Routes}
 */

interface RequestOptions {
  method: string
  headers: Headers
  redirect: 'follow' | 'error' | 'manual'
}

const authLoader = async (): Promise<unknown> => {
  const myHeaders = new Headers()
  myHeaders.append('Authorization', process.env.AUTH_TOKEN!)
  const requestOptions: RequestOptions = {
    method: 'GET',
    headers: myHeaders,
    redirect: 'follow',
  }
  return fetch('/whoami', requestOptions)
    .then((response) => response.text())
    .then((result) => {
      return result
    })
    .catch((error) => {
      return error
    })
}

export default authLoader
