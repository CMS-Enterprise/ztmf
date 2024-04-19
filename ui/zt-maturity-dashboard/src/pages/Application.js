import React, { useState } from "react";
import { useParams } from "react-router-dom";
import { Table, Alert } from "@cmsgov/design-system";
import BackButton from "../components/BackButton";
import THead from "../components/TableHead";
import Title from "../components/ScoreTitle";
import TBody from "../components/TableBody";

export default function Application() {
    const {system} = useParams()
    const [governance, setGovernance] = useState(0.0);
    const [npeAuth, setNpeAuth] = useState(0.0);
    const [threatPrtctn, setThreatPrtctn] = useState(0.0);
    const [appSec, setAppSec] = useState(0.0);
    const [accessability, setAccessability] = useState(0.0);
    const [autoOrch, setAutoOrch] = useState(0.0);
    const [visAnalytics, setVisAnalytics] = useState(0.0);
    const allScores = [governance, npeAuth, threatPrtctn,appSec, accessability, autoOrch, visAnalytics]
    const applicationKeys = ['gov', 'npeAuth','threathPrctn', 'appSec', 'accessability', 'autoOrch', 'visAnalytics']
    const allApplicationNames = ['GOVERNANCE', 'NPE AUTH', 'THREAT PROTECTION', 
                                'APP SECURITY','ACCESSABILITY', 'AUTOMATION & ORCHESTRATION','VISABILITY & ANALYTICS']
    return (
      <>
        <Title system={system} pillar={"Application"} />
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
            <THead pillar={"Application"} />
            {/* <TableBody>{table}</TableBody> */}
            <TBody
              keys={applicationKeys}
              names={allApplicationNames}
              allScores={allScores}></TBody>
          </Table>
        </div>
        <BackButton linkTo={`/Pillars/${system}`}></BackButton>
      </>
    );
}
