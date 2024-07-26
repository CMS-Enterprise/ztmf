import * as React from 'react'
// import Grid from '@mui/material/Grid'
// import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { Container } from '@mui/material'
import { Outlet, useLoaderData } from 'react-router-dom'
import { UsaBanner } from '@cmsgov/design-system'
import { jwtDecode } from 'jwt-decode'
import logo from '../../assets/icons/logo.svg'
import AccountCircleIcon from '@mui/icons-material/AccountCircle'
/**
 * Component that renders the contents of the Dashboard view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */
interface LoaderData {
  email?: string
  name?: string
  preferred_username?: string
  groups?: string[]
}
const Title: React.FC = (): JSX.Element => {
  const loaderData = useLoaderData() as string
  const userInfo = jwtDecode(loaderData) as LoaderData
  return (
    <>
      <UsaBanner />
      <div className="ds-l-row ds-u-margin--0 ds-u-padding-x--2 ds-u-padding-y--0 ds-u-padding-left--6">
        <div className="header-top-wrapper ds-l-md-col--12">
          <div className="region region-cms-header-primary">
            <div className="ds-l-row">
              <div className="ds-l-col--1"></div>
              <div className="ds-l-col--2 ds-u-margin-left--7 ds-u-lg-display--block">
                <img
                  className="ds-u-float--right"
                  src={logo}
                  alt="CMS.gov"
                  width={200}
                  height={100}
                ></img>
              </div>
              <div className="ds-l-col--3 ds-u-lg-display--block ds-u-display--none ds-u-padding-left--0 ds-u-margin-top--5 ds-u-font-weight--semibold">
                Centers for Medicare &amp; Medicaid Services
              </div>
              <div className="ds-l-col--4 ds-u-lg-display--block ds-u-display--none ds-u-padding-left--0 ds-u-margin-top--5 ds-u-font-weight--semibold">
                <div className="ds-u-float--right">
                  <AccountCircleIcon fontSize={'large'} />
                  {userInfo.name ? (
                    <span
                      style={{ verticalAlign: '13px' }}
                      className="ds-text-body--md"
                    >
                      {userInfo.name}
                    </span>
                  ) : (
                    ''
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      {/* </div> */}
      <Container>
        <Typography variant="h3" align="center">
          Zero Trust Maturity Score Dashboard
        </Typography>
        <Outlet />
      </Container>
    </>
  )
}

export default Title
