import React, { useState } from "react";
import { useParams } from "react-router-dom";
import BackButton from "../components/BackButton";
import THead from "../components/TableHead";
import Title from "../components/ScoreTitle";
import TBody from "../components/TableBody";
import {
  Table,
  Alert,
} from "@cmsgov/design-system";

export default function Network(){
    const {system} = useParams()
    const [govCap, setGovCap] = useState(0.0)
    const [netSeg, setNetSeg] = useState(0.0);
    const [threatPrtctn, setThreatPrtctn] = useState(0.0);
    const [encryption, setEncryption] = useState(0.0);
    const [visAnalytics, setVisAnalytics] = useState(0.0)
    const [autoOrch, setAutoOrch] = useState(0.0);
    const allNetworkNames = ['GOVERNANCE CAPABILITY', 'NETWORK SEGMENTATION', 'THREAT PROTECTION', 
                            'ENCRYPTION', 'VISABILITY & ANALYTICS', 'AUTOMATION & ORCHESTRATION']
    const networkKeys = ['govCap', 'netSeg','thrtPrtctn','encryption', 'visAnalytics','autoOrc']
    const allScores = [govCap,netSeg,threatPrtctn,encryption,visAnalytics,autoOrch]
    return (
      <>
        <Title system={system} pillar={"Network"} />
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
            <THead pillar={"Network"} />
            {/* <TableBody>{table}</TableBody> */}
            <TBody keys={networkKeys} names={allNetworkNames} allScores={allScores} ></TBody>
          </Table>
        </div>
        <BackButton linkTo={`/Pillars/${system}`}></BackButton>
      </>
    );
}