import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import FISMASYSTEMQUERY from '../PillarTable/PillarTable.query'
import { fismaSystem } from '../PillarTable/PillarTable.query'
import { useQuery } from '@apollo/client'
import TableTitle from '../TableTitle/TableTitle'
import { Button, Table, TableBody, TableCell, TableRow } from '@mui/material'
// import { Typography } from '@mui/material'
import { Link } from 'react-router-dom'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import PillarTableHead from '../PillarTableHead/PillarTableHead'
/**
 * Component that renders the contents of the Application Page view.
 * @returns {JSX.Element} Component that renders the Application Page contents.
 */

const ApplicationPage: React.FC = (): JSX.Element => {
  const param = useParams<{ systemId?: string }>()
  const id = param.systemId
  const [accessibleApp, setAccessibleApp] = useState<number>(0.0)
  const [accessAuthUser, setAccessAuthUser] = useState<number>(0.0)
  const [secDevDplyWrkFlw, setSecDevDplyWrkFlw] = useState<number>(0.0)
  const [visAnalystics, setVisAnalystics] = useState<number>(0.0)
  const [autoOrch, setAutOrch] = useState<number>(0.0)
  const [secTest, setSecTest] = useState<number>(0.0)
  const [thrtPrctn, setThrtPrctn] = useState<number>(0.0)
  const [governance, setGovernance] = useState<number>(0.0)
  const [fismaSystem, setFismaSystem] = useState<fismaSystem>({
    fismasystemid: '',
    fismauid: '',
    fismaacronym: '',
    functionscores: [],
  })
  const { loading, data } = useQuery(FISMASYSTEMQUERY, { variables: { id } })

  useEffect(() => {
    let isCancelled: boolean = false
    if (data) {
      if (!isCancelled) {
        setFismaSystem(data.fismasystem)
      }
    }
    return () => {
      isCancelled = true
    }
  }, [data])
  useEffect(() => {
    let isCancelled: boolean = false
    if (!isCancelled) {
      if (fismaSystem && fismaSystem.functionscores) {
        fismaSystem.functionscores.forEach((fn) => {
          if (fn.function.pillar === 'Applications') {
            console.log(fn.function.name, fn.score)
            if (fn.function.name === 'AccessibleApplications') {
              setAccessibleApp(fn.score)
            } else if (fn.function.name === 'AccessAuthorization-Users') {
              setAccessAuthUser(fn.score)
            } else if (fn.function.name === 'AppThreatProtection') {
              setThrtPrctn(fn.score)
            } else if (fn.function.name === 'SecureDevDeployWorkflow') {
              setSecDevDplyWrkFlw(fn.score)
            } else if (fn.function.name === 'ApplicationSecurityTesting') {
              setSecTest(fn.score)
            } else if (fn.function.name === 'Application-VisibilityAnalytics') {
              setVisAnalystics(fn.score)
            } else if (
              fn.function.name === 'Application-AutomationOrchestration'
            ) {
              setAutOrch(fn.score)
            } else if (fn.function.name === 'Application-Governance') {
              setGovernance(fn.score)
            }
          }
        })
      }
    }
    return () => {
      isCancelled = true
    }
  }, [fismaSystem])
  return (
    <>
      {loading ? (
        <p>Loading ...</p>
      ) : (
        <>
          <TableTitle
            system={fismaSystem.fismaacronym}
            pillarType="Application"
          />
          <Table sx={{ minWidth: 650, border: 1 }}>
            <PillarTableHead />
            <TableBody>
              {/* Accessible Applications */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  ACCESSABILITY
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {accessibleApp}
                </TableCell>
              </TableRow>
              {/* AccessAuthorization-Users */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  ACCESS AUTHORIZATION-USERS
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {accessAuthUser}
                </TableCell>
              </TableRow>
              {/* Automation Orchestration */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  AUTOMATION & ORCHESTRATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {autoOrch}
                </TableCell>
              </TableRow>
              {/* Governance */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  GOVERNANCE
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {governance}
                </TableCell>
              </TableRow>
              {/* SECURE DEVELOPER DEPLOY WORKFLOW */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  SECURE DEVELOPER DEPLOY WORKFLOW
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {secDevDplyWrkFlw}
                </TableCell>
              </TableRow>
              {/* Security Testing */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  SECURITY TESTING
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {secTest}
                </TableCell>
              </TableRow>
              {/* Threat Protection */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  THREAT PROTECTION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {thrtPrctn}
                </TableCell>
              </TableRow>
              {/* Visbility & Analytics */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  VISIBILITY & ANALYTICS
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {visAnalystics}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <Link to={`/pillars/${id}`}>
            <Button
              variant="contained"
              sx={{ mt: 10, ml: 1 }}
              endIcon={<ArrowBackIcon />}
            >
              Back
            </Button>
          </Link>
        </>
      )}
    </>
  )
}

export default ApplicationPage
