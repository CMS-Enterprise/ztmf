import * as React from 'react'
// import Grid from '@mui/material/Grid'
// import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { Container } from '@mui/material'
import { Outlet } from 'react-router-dom'
/**
 * Component that renders the contents of the Dashboard view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */

const Title: React.FC = (): JSX.Element => {
    return (
      <>
        <Container maxWidth="md">
          <Typography variant="h3" align='center'>
            Zero Trust Maturity Score Dashboard
          </Typography>
          <Outlet />
        </Container>
      </>
    )
}

export default Title