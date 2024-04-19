import React, { useState } from "react";
import { useParams } from "react-router-dom";
import { Table, Alert } from "@cmsgov/design-system";
import BackButton from "../components/BackButton";
import THead from "../components/TableHead";
import Title from "../components/ScoreTitle";
import TBody from "../components/TableBody";

export default function Data() {
    const { system } = useParams();
    const [governance, setGovernance] = useState(0.0);
    const [invMgnmnt, setInvMgnmnt] = useState(0.0);
    const [accessDet, setAccessDet] = useState(0.0);
    const [logging, setLogging] = useState(0.0);
    const [visAnalytics, setVisAnalytics] = useState(0.0);
    const [autoOrch, setAutoOrch] = useState(0.0);
    const allScores = [governance, invMgnmnt, accessDet, logging, visAnalytics, autoOrch];
    const dataKeys = ['governance', 'invMgnmnt', 'accessDet', 'logging', 'visAnalytics', 'autoOrch']
    const allDataNames = ['GOVERNANCE', 'INVENTORY MANAGEMENT', 'ACCESS DETERMINATION', 
                        'LOGGING', 'VISUAL ANALYTICS', 'AUTOMATION & ORCHESTRATION']
    return (
      <>
        <Title system={system} pillar={"Data"} />
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
            <THead pillar={"Data"} />
            <TBody
              keys={dataKeys}
              names={allDataNames}
              allScores={allScores}>
            
            </TBody>
          </Table>
        </div>
        <BackButton linkTo={`/Pillars/${system}`}></BackButton>
      </>
    );
}
