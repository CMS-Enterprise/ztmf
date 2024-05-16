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

const DataPage: React.FC = (): JSX.Element => {
  const param = useParams<{ systemId?: string }>()
  const id = param.systemId
  const [availability, setAvailability] = useState<number>(0.0)
  const [governance, setGovernance] = useState<number>(0.0)
  const [invMngmnt, setinvMngmnt] = useState<number>(0.0)
  const [categorization, setCategorization] = useState<number>(0.0)
  const [access, setAccess] = useState<number>(0.0)
  const [encryption, setEncryption] = useState<number>(0.0)
  const [visAnalystics, setVisAnalystics] = useState<number>(0.0)
  const [autoOrch, setAutOrch] = useState<number>(0.0)
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
          if (fn.function.pillar === 'Data') {
            if (fn.function.name === 'DataInventoryManagement') {
              setinvMngmnt(fn.score)
            } else if (fn.function.name === 'DataCategorization') {
              setCategorization(fn.score)
            } else if (fn.function.name === 'DataAvailability') {
              setAvailability(fn.score)
            } else if (fn.function.name === 'DataAccess') {
              setAccess(fn.score)
            } else if (fn.function.name === 'DataEncryption') {
              setEncryption(fn.score)
            } else if (fn.function.name === 'Data-VisibilityAnalytics') {
              setVisAnalystics(fn.score)
            } else if (fn.function.name === 'Data-AutomationOrchestration') {
              setAutOrch(fn.score)
            } else if (fn.function.name === 'Data-Governance') {
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
          <TableTitle system={fismaSystem.fismaacronym} pillarType="Data" />
          <Table sx={{ minWidth: 650, border: 1 }}>
            <PillarTableHead />
            <TableBody>
              {/* Access */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  ACCESS DETERMINATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {access}
                </TableCell>
              </TableRow>
              {/* Automation & Orchestration */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  AUTOMATION & ORCHESTRATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {autoOrch}
                </TableCell>
              </TableRow>
              {/* Availability */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  AVAILABILITY
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {availability}
                </TableCell>
              </TableRow>
              {/* Categorization */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  CATEGORIZATION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {categorization}
                </TableCell>
              </TableRow>
              {/* Encryption */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  ENCRYPTION
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {encryption}
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
              {/* Inventory Management */}
              <TableRow>
                <TableCell sx={{ border: 1 }} align="center">
                  INVENTORY MANAGEMENT
                </TableCell>
                <TableCell sx={{ border: 1 }} align="center">
                  {invMngmnt}
                </TableCell>
              </TableRow>
              {/* Visibility & Analytics */}
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
export default DataPage
