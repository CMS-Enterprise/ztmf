import * as React from 'react'
import { Table, TableBody, TableCell, TableRow, TableHead } from '@mui/material'
const ScoreTable: React.FC = (): JSX.Element => {
  return (
    <>
      <Table
        sx={{ minWidth: 650, maxHeight: 10, border: 1, mt: 8 }}
        size="small"
      >
        <TableHead>
          <TableRow>
            <TableCell
              sx={{
                border: 1,
                backgroundColor: '#DCDCDC',
              }}
              align="center"
            >
              Traditional
            </TableCell>
            <TableCell
              sx={{
                border: 1,
                backgroundColor: '#DCDCDC',
              }}
              align="center"
            >
              Initial
            </TableCell>
            <TableCell
              sx={{
                border: 1,
                backgroundColor: '#DCDCDC',
              }}
              align="center"
            >
              Advanced
            </TableCell>
            <TableCell
              sx={{
                border: 1,
                backgroundColor: '#DCDCDC',
              }}
              align="center"
            >
              Optimal
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          <TableRow>
            <TableCell
              sx={{
                border: 1,
              }}
              align="center"
            >
              1-1.74
            </TableCell>
            <TableCell
              sx={{
                border: 1,
              }}
              align="center"
            >
              1.75-2.74
            </TableCell>
            <TableCell
              sx={{
                border: 1,
              }}
              align="center"
            >
              2.75-3.65
            </TableCell>
            <TableCell sx={{ border: 1 }} align="center">
              3.66+
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </>
  )
}

export default ScoreTable
