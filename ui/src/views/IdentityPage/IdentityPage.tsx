import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import FISMASYSTEMQUERY from '../PillarTable/PillarTable.query'
import { fismaSystem } from '../PillarTable/PillarTable.query'
import { useQuery } from '@apollo/client'
import TableTitle from '../TableTitle/TableTitle'
import PillarTableHead from '../PillarTableHead/PillarTableHead'
import { Button, Table, TableBody, TableCell, TableRow } from '@mui/material'
// import { Typography } from '@mui/material'
import { Link } from 'react-router-dom'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'

/**
 * Component that renders the contents of the Identity Page view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */

const IdentityPage: React.FC = (): JSX.Element => {
  const param = useParams<{ systemId?: string }>()
  const id = param.systemId
  const [authUser, setAuthUser] = useState<number>(0.0)
  const [idStoreUser, setIdStoreUser] = useState<number>(0.0)
  const [visAnalystics, setVisAnalystics] = useState<number>(0.0)
  const [autoOrch, setAutoOrch] = useState<number>(0.0)
  const [governance, setGovernance] = useState<number>(0.0)
  const [accMgnt, setAccMgnt] = useState<number>(0.0)
  const [riskAssmnt, setRiskAssmnt] = useState<number>(0.0)
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
          if (fn.function.pillar === 'Identity') {
            if (fn.function.name === 'Authentication-Users') {
              setAuthUser(fn.score)
            } else if (fn.function.name === 'IdentityStores-Users') {
              setIdStoreUser(fn.score)
            } else if (fn.function.name === 'Identity-VisibilityAnalytics') {
              setVisAnalystics(fn.score)
            } else if (
              fn.function.name === 'Identity-AutomationOrchestration'
            ) {
              setAutoOrch(fn.score)
            } else if (fn.function.name === 'Identity-Governance') {
              setGovernance(fn.score)
            } else if (fn.function.name === 'AccessManagement') {
              setAccMgnt(fn.score)
            } else if (fn.function.name === 'RiskAssessment') {
              setRiskAssmnt(fn.score)
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
          <TableTitle system={fismaSystem.fismaacronym} pillarType="Identity" />
          <Table sx={{ minWidth: 650, border: 1 }}>
            <PillarTableHead />
            <TableBody>
              {/* Access Management */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  ACCESS MANAGEMENT
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {accMgnt}
                </TableCell>
              </TableRow>
              {/* Automation & orchestration */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  AUTOMATION & ORCHESTRATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {autoOrch}
                </TableCell>
              </TableRow>
              {/* Governance */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  GOVERNANCE CAPABILITY
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {governance}
                </TableCell>
              </TableRow>
              {/* Identity stores */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  IDENTITY STORES
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {idStoreUser}
                </TableCell>
              </TableRow>
              {/* Risk Assessment */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  RISK ASSESSMENT
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {riskAssmnt}
                </TableCell>
              </TableRow>
              {/* User Auth */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  USER AUTH
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {authUser}
                </TableCell>
              </TableRow>
              {/* visability & analytics */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  VISABILITY & ANALYTICS
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
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

export default IdentityPage
