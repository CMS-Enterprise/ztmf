import React,{useState} from "react";
import { Link, useParams, Outlet} from "react-router-dom";
import {Table, Alert, TableCell, TableBody, TableHead, TableRow} from '@cmsgov/design-system'
import BackButton from '../components/BackButton'

export default function Pillars() {
  const {fismaSystem} = useParams();
  const [path, setPath] = useState("")
  const [identity, setIdentity] = useState(0.0)
  const [devices, setDevices] = useState(0.0)
  const [network, setNetwork] = useState(0.0);
  const [application, setApplication] = useState(0.0);
  const [data, setData] = useState(0.0);
  const pillarKeys = ['Identity', 'Devices', 'Network', 'Application', 'Data']
  const allScores = [identity, devices, network, application, data]
  const table = pillarKeys.map((pillar, index) => {
    return (
      <TableRow key={`pillar_${pillar}`}>
        <TableCell
          key={`${pillar}_name`}
          headers={`${pillar}_header`}
          stackedTitle={`${pillar}_stackedTitle`}
          align="center">
          <strong>{pillar.toUpperCase()}</strong>
        </TableCell>
        <TableCell
          key={`${pillar}_score`}
          headers={`${pillar}_Header_Score`}
          stackedTitle={`${pillar}_Score_StackedTitle`}
          align="center">
          {allScores[index]}
        </TableCell>
        <TableCell
          key={`${pillar}_link`}
          headers={`${pillar}_Headers_link`}
          stackedTitle={`${pillar}_Link_StackedTitle`}
          align="center">
          <Link to={`/Pillars/${fismaSystem}/${pillar}`}>
            CLICK FOR FUNCTIONS
          </Link>
        </TableCell>
      </TableRow>
    );
  })
  return (
    <>
      {/* Title */}
      <div className="ds-u-justify-content--center">
        <h1 className="ds-u-md-text-align--center  ds-text-heading--3xl ds-u-margin-top--7 ds-u-margin-bottom--7">
          {fismaSystem} Maturity Score Pillars
        </h1>
      </div>
      {/* Table */}
      <div className="ds-u-display--flex ds-u-justify-content--center">
        <Table
          scrollableNotice={
            <Alert className="ds-c-table__scroll-alert" role="status">
              <p class="ds-c-alert__text" className="ds-c-alert__text">
                Scroll using arrow keys to see more
              </p>
            </Alert>
          }
          stackable
          stackableBreakpoint="md">
          <TableHead>
            <TableRow>
              <TableCell key="Pillars" id="pillars" align="center">
                Pillars
              </TableCell>
              <TableCell key="Score" id="score" align="center">
                Score
              </TableCell>
              <TableCell key="Function" id="function" align="center">
                Function
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {table}
          </TableBody>
        </Table>
      </div>
      {/* Back Button */}
      <BackButton linkTo={"/"}></BackButton>
      <Outlet />
    </>
  );
}