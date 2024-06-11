// import {useState} from 'react';
import { useState, useEffect } from 'react'
import { useQuery } from '@apollo/client'
import Typography from '@mui/material/Typography'
import FISMASYSTEMQUERY from './PillarTable.query'
import { fismaSystem } from './PillarTable.query'
import { Link, useLocation, useParams, Location } from 'react-router-dom'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { Routes, RouteIds } from '@/router/constants'
import {
  Button,
  Table,
  TableBody,
  TableCell,
  TableRow,
  TableHead,
} from '@mui/material'
import ScoreTable from '../ScoreTable/ScoreTable'
import TableTitle from '../TableTitle/TableTitle'
/**
 * Component that renders the contents of the Dashboard view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */

const PillarPage: React.FC = (): JSX.Element => {
  const param = useParams<{ systemId?: string }>()
  const location: Location = useLocation()
  const id = param.systemId
  const { loading, data } = useQuery(FISMASYSTEMQUERY, { variables: { id } })
  const [avgIdentityScore, setAvgIdentityScore] = useState<number>(0.0)
  const [avgDevicesScore, setAvgDevicesScore] = useState<number>(0.0)
  const [avgNetworkScore, setAvgNetworkScore] = useState<number>(0.0)
  const [avgApplicationScore, setAvgApplicationScore] = useState<number>(0.0)
  const [avgDataScore, setAvgDataScore] = useState<number>(0.0)
  const [fismaSystem, setFismaSystem] = useState<fismaSystem>({
    fismasystemid: '',
    fismauid: '',
    fismaacronym: '',
    functionscores: [],
  })
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
        let identityScore: number = 0
        let identityCount: number = 0
        let devicesScore: number = 0
        let devicesCount: number = 0
        let networksScore: number = 0
        let networksCount: number = 0
        let applicationScore: number = 0
        let applicationCount: number = 0
        let dataScore: number = 0
        let dataCount: number = 0
        fismaSystem.functionscores.forEach((fn) => {
          if (fn.function.pillar === 'Identity') {
            identityScore += fn.score
            identityCount += 1
          } else if (fn.function.pillar === 'Devices') {
            devicesScore += fn.score
            devicesCount += 1
          } else if (fn.function.pillar === 'Networks') {
            networksScore += fn.score
            networksCount += 1
          } else if (fn.function.pillar === 'Applications') {
            applicationCount += 1
            applicationScore += fn.score
          } else if (fn.function.pillar === 'Data') {
            dataScore += fn.score
            dataCount += 1
          }
        })
        const identityAvg: number =
          Number((identityScore / identityCount).toFixed(2)) || 0
        const devicesAvg: number =
          Number((devicesScore / devicesCount).toFixed(2)) || 0
        const networksAvg: number =
          Number((networksScore / networksCount).toFixed(2)) || 0
        const applicationAvg: number =
          Number((applicationScore / applicationCount).toFixed(2)) || 0
        const dataAvg: number = Number((dataScore / dataCount).toFixed(2)) || 0
        setAvgIdentityScore(identityAvg)
        setAvgDevicesScore(devicesAvg)
        setAvgNetworkScore(networksAvg)
        setAvgApplicationScore(applicationAvg)
        setAvgDataScore(dataAvg)
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
          <TableTitle system={fismaSystem.fismaacronym} pillarType="" />
          <Table sx={{ minWidth: 650, border: 1 }}>
            <TableHead>
              <TableRow>
                <TableCell
                  key="Pillar title"
                  id="pillarTitle"
                  sx={{ border: 1, backgroundColor: '#DCDCDC' }}
                  align="center"
                  size="small"
                >
                  <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                    Pillars
                  </Typography>
                </TableCell>
                <TableCell
                  key="Description"
                  id="description"
                  sx={{ border: 1, backgroundColor: '#DCDCDC' }}
                  align="center"
                  size="small"
                >
                  <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                    Score
                  </Typography>
                </TableCell>
                <TableCell
                  key="Links"
                  id="links"
                  sx={{ border: 1, backgroundColor: '#DCDCDC' }}
                  align="center"
                  size="small"
                >
                  <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                    Functions
                  </Typography>
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow key="identity pillar">
                <TableCell
                  key="identity pillar cell"
                  headers="identityPillar"
                  sx={{ border: 1 }}
                  align="center"
                >
                  IDENTITY
                </TableCell>
                <TableCell
                  key="identity pillar score"
                  headers="idenityScore"
                  sx={{ border: 1 }}
                  align="center"
                >
                  {avgIdentityScore}
                </TableCell>
                <TableCell
                  key="idenity-pillar-link"
                  headers="identity-link"
                  sx={{ border: 1, p: 0 }}
                  align="center"
                >
                  <Link to={`${location.pathname}/${RouteIds.IDENTITY}`}>
                    <p>Click for Functions</p>
                  </Link>
                </TableCell>
              </TableRow>
              <TableRow key="devices pillar">
                <TableCell
                  key="devices pillar cell"
                  headers="devicesPillar"
                  sx={{ border: 1 }}
                  align="center"
                >
                  DEVICES
                </TableCell>
                <TableCell
                  key="devices score cell"
                  headers="devices"
                  sx={{ border: 1 }}
                  align="center"
                >
                  {avgDevicesScore}
                </TableCell>
                <TableCell
                  key="devices links cell"
                  headers="devicesLinks"
                  sx={{ border: 1, p: 0 }}
                  align="center"
                >
                  <Link to={`${location.pathname}/${RouteIds.DEVICES}`}>
                    <p>Click for Functions</p>
                  </Link>
                </TableCell>
              </TableRow>
              <TableRow key="networks pillar">
                <TableCell
                  sx={{ border: 1 }}
                  key="network pillar cell"
                  headers="networkPillar"
                  align="center"
                >
                  NETWORK
                </TableCell>
                <TableCell
                  key="network pillar score"
                  headers="networkScore"
                  sx={{ border: 1 }}
                  align="center"
                >
                  {avgNetworkScore}
                </TableCell>
                <TableCell
                  key="network-pillar-link"
                  headers="network-link"
                  sx={{ border: 1, p: 0 }}
                  align="center"
                >
                  <Link to={`${location.pathname}/${RouteIds.NETWORKS}`}>
                    <p>Click for Functions</p>
                  </Link>
                </TableCell>
              </TableRow>
              <TableRow key="application pillar">
                <TableCell
                  key="application pillar cell"
                  headers="applicationPillar"
                  sx={{ border: 1 }}
                  align="center"
                >
                  APPLICATION
                </TableCell>
                <TableCell
                  key="application pillar score"
                  headers="applicationScore"
                  sx={{ border: 1 }}
                  align="center"
                >
                  {avgApplicationScore}
                </TableCell>
                <TableCell
                  key="application-pillar-link"
                  headers="application-link"
                  sx={{ border: 1, p: 0 }}
                  align="center"
                >
                  <Link to={`${location.pathname}/${RouteIds.APPLICATIONS}`}>
                    <p>Click for Functions</p>
                  </Link>
                </TableCell>
              </TableRow>
              <TableRow key="data pillar">
                <TableCell
                  sx={{ border: 1 }}
                  key="data pillar cell"
                  headers="dataPillar"
                  align="center"
                >
                  DATA
                </TableCell>
                <TableCell
                  sx={{ border: 1 }}
                  key="data pillar score"
                  headers="dataScore"
                  align="center"
                >
                  {avgDataScore}
                </TableCell>
                <TableCell
                  sx={{ border: 1, p: 0 }}
                  key="data-pillar-link"
                  headers="data-link"
                  align="center"
                >
                  <Link to={`${location.pathname}/${RouteIds.DATA}`}>
                    <p>Click for Functions</p>
                  </Link>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
          <ScoreTable />
          <Link to={Routes.ROOT}>
            <Button
              variant="contained"
              sx={{ mt: 10, ml: 1, marginLeft: 0, marginTop: 4 }}
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

export default PillarPage
