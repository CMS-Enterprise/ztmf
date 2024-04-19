import React from "react";
import {TableHead, TableCell, TableRow} from '@cmsgov/design-system'

export default function THead({pillar}) {
    return (
      <TableHead>
        <TableRow>
          <TableCell
            key={`${pillar}_head`}
            id={`${pillar}_head`}
            align="center">
            {pillar.toUpperCase()}
          </TableCell>
          <TableCell key="Score" id="score" align="center">
            SCORE
          </TableCell>
        </TableRow>
      </TableHead>
    );
}