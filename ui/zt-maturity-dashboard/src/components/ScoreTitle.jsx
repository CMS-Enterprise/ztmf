import React from "react";

export default function Title({system, pillar}) {
    return (
      <div className="ds-u-justify-content--center">
        <h1 className="ds-u-md-text-align--center  ds-text-heading--3xl ds-u-margin-top--7 ds-u-margin-bottom--7">
          {system} {pillar} Pillar Score Functions
        </h1>
      </div>
    );

}