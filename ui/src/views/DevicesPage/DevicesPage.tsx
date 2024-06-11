import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import FISMASYSTEMQUERY, {
  fismaSystem,
  determineLevel,
  functionscores,
} from '../PillarTable/PillarTable.query'
import { useQuery } from '@apollo/client'
import TableTitle from '../TableTitle/TableTitle'
import {
  Box,
  Button,
  Table,
  TableBody,
  TableCell,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import { Link } from 'react-router-dom'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import PillarTableHead from '../PillarTableHead/PillarTableHead'
/**
 * Component that renders the contents of the Dashboard view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */

type dataPillarFunction = functionscores[]
const DevicesPage: React.FC = (): JSX.Element => {
  const param = useParams<{ systemId?: string }>()
  const id = param.systemId
  const [fismaDevicePillars, setFismaDevicePillars] = useState<
    functionscores[]
  >([])
  const [fismaSystem, setFismaSystem] = useState<fismaSystem>({
    fismasystemid: '',
    fismauid: '',
    fismaacronym: '',
    functionscores: [],
  })
  const pillarNameMap: { [key: string]: string } = {
    AssetRiskManagement: 'ASSET RISK MANAGEMENT',
    'Device-AutomationOrchestration': 'AUTOMATION & ORCHESTRATION',
    'Device-Governance': 'GOVERNANCE',
    DeviceThreatProtection: 'THREAT PROTECTION',
    PolicyEnforcement: 'POLICY ENFORCEMENT',
    ResourceAccess: 'RESOURCE ACCESS',
    'Device-VisibilityAnalytics': 'VISIBILITY ANALYTICS',
  }
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
    const order: string[] = [
      'AssetRiskManagement',
      'Device-AutomationOrchestration',
      'Device-Governance',
      'DeviceThreatProtection',
      'PolicyEnforcement',
      'Device-VisibilityAnalytics',
      'ResourceAccess',
    ]
    function customSort(a: functionscores, b: functionscores) {
      return order.indexOf(a.function.name) - order.indexOf(b.function.name)
    }
    let isCancelled: boolean = false
    if (!isCancelled) {
      if (fismaSystem && fismaSystem.functionscores) {
        const dataArr: dataPillarFunction = []
        fismaSystem.functionscores.forEach((fn) => {
          if (fn.function.pillar === 'Devices') {
            dataArr.push(fn)
          }
        })
        const sortedDataArr = dataArr.slice().sort(customSort)
        setFismaDevicePillars((prev) => [...prev, ...sortedDataArr])
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
              {fismaDevicePillars.map((fn, idx) => {
                const [fnLevel, description] = determineLevel(
                  fn.score,
                  fn.function
                )
                return (
                  <TableRow key={idx}>
                    <TableCell
                      key={fn.function.name}
                      sx={{ border: 1 }}
                      align="center"
                    >
                      <Box display="flex" justifyContent="flex-end">
                        <sub>
                          <Tooltip
                            title={fn.function.description}
                            placement="top"
                          >
                            <InfoOutlinedIcon
                              color="primary"
                              shapeRendering="geometricPrecision"
                              fontSize="small"
                            />
                          </Tooltip>
                        </sub>
                      </Box>
                      <Box sx={{ fontWeight: 'bold' }}>
                        {pillarNameMap[fn.function.name]}
                      </Box>
                    </TableCell>
                    <TableCell sx={{ border: 1 }} align="center">
                      <Box
                        display="flex"
                        justifyContent="center"
                        alignItems="center"
                      >
                        <Typography component="span" fontWeight="bold">
                          {fnLevel}
                        </Typography>
                        <Typography component="span"> (</Typography>
                        <Typography component="span" fontWeight="bold">
                          {fn.score}
                        </Typography>
                        <Typography component="span"> )</Typography>
                      </Box>
                      <Box sx={{ textTransform: 'capitalize' }}>
                        {description}
                      </Box>
                    </TableCell>
                    <TableCell sx={{ border: 1 }} align="center">
                      {fn.notes}
                    </TableCell>
                  </TableRow>
                )
              })}
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
