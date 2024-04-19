import React, {useState} from "react";
import {useParams} from 'react-router-dom'
import BackButton from "../components/BackButton";
import THead from "../components/TableHead";
import Title from "../components/ScoreTitle";
import {
  Table,
  Alert,
} from "@cmsgov/design-system";
import TBody from "../components/TableBody";

export default function Identity() {
    const {system} = useParams()
    const [storeScore, setStoreScore] = useState(0.0)
    const [userAuth, setUserAuth] = useState(0.0)
    const [deviceAuth, setDeviceAuth] =  useState(0.0)
    const [visAnalytics, setVisAnalytics] = useState(0.0)
    const [autoOrch, setAutoOrch] = useState(0.0)
    const [userId, setUserId] = useState(0.0)
    const [deviceId, setDeviceId] = useState(0.0)
    const allIdentityNames = ["IDENTITY STORES", 'USER AUTH', 'DEVICE AUTH', 'VISABILITY & ANALYTICS',
                            'AUTOMATION & ORCHESTRATION', 'USER IDENTITIES', 'DEVICE IDENTITIES']
    const allScores = [storeScore, userAuth, deviceAuth, visAnalytics, autoOrch, userId, deviceId]
    const keyNames = ['stores', 'user_auth', 'device_aut', 'v_a', 'a_o', 'user_identities', 'device_identities']

    return (
      <div>
        {/* Title */}
        <Title system={system} pillar={'Identity'}/>
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
            <THead pillar={'identity'}/>
            <TBody keys={keyNames} allScores={allScores} names={allIdentityNames}></TBody>
          </Table>
        </div>
        <BackButton linkTo={`/Pillars/${system}`}></BackButton>
      </div>
    );
}