import React, { useState } from "react";
import { useParams } from "react-router-dom";
import BackButton from "../components/BackButton";
import THead from "../components/TableHead";
import Title from "../components/ScoreTitle";
import {
  Table,
  Alert
} from "@cmsgov/design-system";
import TBody from "../components/TableBody";


export default function Devices() {
    const {system} = useParams()
    const [govCap, setGovCap] = useState(0.0)
    const [complianceMonitoring, setComplianceMonitoring] = useState(0.0)
    const [dataAccess, setDataAccess] = useState(0.0)
    const [assetManagement, setAssetManagement] = useState(0.0)
    const [visAnalytics, setVisAnalytics] = useState(0.0);
    const allDeviceNames = ['GOVERNANCE CAPABILITY', 'COMPLIANCE MONITORING', 'DATA ACCESS', 'ASSET MANAGEMENT', 'VISABILITY & ANALYTICS']
    const deviceKeys = ['gov_cap','comp_mon', 'data_access', 'asset_mngmt','vis_analytics']
    const allScores = [govCap,complianceMonitoring,dataAccess,assetManagement,visAnalytics]
    return (
      <>
        <Title system={system} pillar={"Devices"} />
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
            <THead pillar={"identity"} />
            {/* <TableBody>{table}</TableBody> */}
            <TBody keys={deviceKeys} names={allDeviceNames} allScores={allScores}></TBody>
          </Table>
        </div>
        <BackButton linkTo={`/Pillars/${system}`}></BackButton>
      </>
    );
}