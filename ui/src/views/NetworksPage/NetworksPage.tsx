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

const NetworksPage: React.FC = (): JSX.Element => {
  const param = useParams<{ systemId?: string }>()
  const id = param.systemId
  const [fismaSystem, setFismaSystem] = useState<fismaSystem>({
    fismasystemid: '',
    fismauid: '',
    fismaacronym: '',
    functionscores: [],
  })
  const [autoOrch, setAutOrch] = useState<number>(0.0)
  const [encrytion, setEncryption] = useState<number>(0.0)
  const [resilience, setResilience] = useState<number>(0.0)
  const [segmentation, setSegmentation] = useState<number>(0.0)
  const [trafficMngt, setTrafficMngt] = useState<number>(0.0)
  const [governance, setGovernance] = useState<number>(0.0)
  const [visAnalystics, setVisAnalystics] = useState<number>(0.0)
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
          if (fn.function.pillar === 'Networks') {
            if (fn.function.name === 'Network-AutomationOrchestration') {
              setAutOrch(fn.score)
            } else if (fn.function.name === 'Network-Encryption') {
              setEncryption(fn.score)
            } else if (fn.function.name === 'Network-Governance') {
              setGovernance(fn.score)
            } else if (fn.function.name === 'NetworkResilience') {
              setResilience(fn.score)
            } else if (fn.function.name === 'NetworkSegmentation') {
              setSegmentation(fn.score)
            } else if (fn.function.name === 'NetworkTrafficManagement') {
              setTrafficMngt(fn.score)
            } else if (fn.function.name === 'Network-VisibilityAnalytics') {
              setVisAnalystics(fn.score)
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
          <TableTitle system={fismaSystem.fismaacronym} pillarType="Network" />
          <Table sx={{ minWidth: 650, border: 1 }}>
            <PillarTableHead />
            <TableBody>
              {/* Automation Orchestration */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  AUTOMATION & ORCHESTRATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {autoOrch}
                </TableCell>
              </TableRow>
              {/* Encryption */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  ENCRYPTION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {encrytion}
                </TableCell>
              </TableRow>
              {/* Governance */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  GOVERNANCE CAPABILITY
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {governance}
                </TableCell>
              </TableRow>
              {/* Resilience */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  RESILIENCE
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {resilience}
                </TableCell>
              </TableRow>
              {/* Segmentation */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  NETWORK SEGMENTATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {segmentation}
                </TableCell>
              </TableRow>
              {/* Traffic Management */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  TRAFFIC MANAGEMENT
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {trafficMngt}
                </TableCell>
              </TableRow>
              {/* Visibility Analytics */}
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

export default NetworksPage
