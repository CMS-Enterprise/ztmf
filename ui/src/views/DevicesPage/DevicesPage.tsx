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
 * Component that renders the contents of the Dashboard view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */

const DevicesPage: React.FC = (): JSX.Element => {
  const param = useParams<{ systemId?: string }>()
  const id = param.systemId
  const [asstRiskAssmnt, setAsstRiskAssmnt] = useState<number>(0.0)
  const [autOrch, setAutOrch] = useState<number>(0.0)
  const [governance, setGov] = useState<number>(0.0)
  const [thrtPrtctn, setThrtPrtctn] = useState<number>(0.0)
  const [visAnalystics, setVisAnalystics] = useState<number>(0.0)
  const [policyEnforcement, setPolicyEnforcment] = useState<number>(0.0)
  const [resourceAccess, setResourceAccess] = useState<number>(0.0)
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
          if (fn.function.pillar === 'Devices') {
            if (fn.function.name === 'AssetRiskManagement') {
              setAsstRiskAssmnt(fn.score)
            } else if (fn.function.name === 'Device-AutomationOrchestration') {
              setAutOrch(fn.score)
            } else if (fn.function.name === 'Device-Governance') {
              setGov(fn.score)
            } else if (fn.function.name === 'DeviceThreatProtection') {
              setThrtPrtctn(fn.score)
            } else if (fn.function.name === 'Device-VisibilityAnalytics') {
              setVisAnalystics(fn.score)
            } else if (fn.function.name === 'PolicyEnforcement') {
              setPolicyEnforcment(fn.score)
            } else if (fn.function.name === 'ResourceAccess') {
              setResourceAccess(fn.score)
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
        <p> Loading ...</p>
      ) : (
        <>
          <TableTitle system={fismaSystem.fismaacronym} pillarType="Device" />
          <Table sx={{ minWidth: 650, border: 1 }}>
            <PillarTableHead />
            <TableBody>
              {/* Asset Risk Management */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  ASSET RISK MANAGEMENT
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {asstRiskAssmnt}
                </TableCell>
              </TableRow>
              {/* Device-AutomationOrchestration */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  AUTOMATION ORCHESTRATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {autOrch}
                </TableCell>
              </TableRow>
              {/* Device-Governance */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  GOVERNANCE
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {governance}
                </TableCell>
              </TableRow>
              {/* Policy Enforcment */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  POLICY ENFORCEMENT
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {policyEnforcement}
                </TableCell>
              </TableRow>
              {/* Resource Access */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  RESOURCE ACCESS
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {resourceAccess}
                </TableCell>
              </TableRow>
              {/* Threat Protection */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  THREAT PROTECTION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  {thrtPrtctn}
                </TableCell>
              </TableRow>
              {/* Visibility Analytics */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center" size="medium">
                  VISIBILITY ANALYTICS
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

export default DevicesPage
