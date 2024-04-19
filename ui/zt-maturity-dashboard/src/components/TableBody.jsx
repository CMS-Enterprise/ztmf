import React from "react";
import {
  Table,
  Alert,
  TableCell,
  TableBody,
  TableRow,
} from "@cmsgov/design-system";


export default function TBody({keys, names, allScores}) {
    const table = keys.map((key, index) => {
        return (
        <TableRow key={`identity_${key}`}>
            <TableCell
            key={`identity_${key}_name`}
            headers={`identity_${key}_name`}
            stackedTitle={`identity_${key}_name`}
            align="center">
            {names[index]}
            </TableCell>
            <TableCell
            key={`identity_${key}_score`}
            headers={`identity_${key}_score`}
            stackedTitle={`identity_${key}_score`}
            align="center">
            {allScores[index]}
            </TableCell>
        </TableRow>
        );
    });
    return table
}