import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import FISMASYSTEMQUERY, {
  functionscores,
  fismaSystem,
  determineLevel,
} from '../PillarTable/PillarTable.query'
import { useQuery } from '@apollo/client'
import TableTitle from '../TableTitle/TableTitle'
import PillarTableHead from '../PillarTableHead/PillarTableHead'
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

// import { Typography } from '@mui/material'
import { Link } from 'react-router-dom'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'

/**
 * Component that renders the contents of the Identity Page view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */
type dataPillarFunction = functionscores[]
const IdentityPage: React.FC = (): JSX.Element => {
  const pillarNameMap: { [key: string]: string } = {
    AccessManagement: 'ACCESS MANAGEMENT',
    'Identity-AutomationOrchestration': 'AUTOMATION & ORCHESTRATION',
    'Identity-Governance': 'GOVERNANCE CAPABILITY',
    'IdentityStores-Users': 'IDENTITY STORES',
    RiskAssessment: 'RISK ASSESSMENT',
    'Authentication-Users': 'USER AUTH',
    'Identity-VisibilityAnalytics': 'VISABILITY & ANALYTICS',
  }
  const param = useParams<{ systemId?: string }>()
  const id = param.systemId
  const [fismaIdentityPillars, setFismaIdentityPillars] = useState<
    functionscores[]
  >([])
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
    const order: string[] = [
      'AccessManagement',
      'Identity-AutomationOrchestration',
      'Identity-Governance',
      'IdentityStores-Users',
      'RiskAssessment',
      'Authentication-Users',
      'Identity-VisibilityAnalytics',
    ]
    function customSort(a: functionscores, b: functionscores) {
      return order.indexOf(a.function.name) - order.indexOf(b.function.name)
    }
    let isCancelled: boolean = false
    if (!isCancelled) {
      if (fismaSystem && fismaSystem.functionscores) {
        const dataArr: dataPillarFunction = []
        fismaSystem.functionscores.forEach((fn) => {
          if (fn.function.pillar === 'Identity') {
            dataArr.push(fn)
          }
        })
        const sortedDataArr = dataArr.slice().sort(customSort)
        setFismaIdentityPillars((prev) => [...prev, ...sortedDataArr])
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
          <Table sx={{ border: 1 }}>
            <PillarTableHead />
            <TableBody>
              {fismaIdentityPillars.map((fn, idx) => {
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
              sx={{
                mt: 10,
                ml: 1,
                marginLeft: 0,
                marginTop: 4,
                marginBottom: 10,
              }}
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
