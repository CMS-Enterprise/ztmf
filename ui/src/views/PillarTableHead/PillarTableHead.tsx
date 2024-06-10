import * as React from 'react'
import { TableHead, TableRow, TableCell } from '@mui/material'
import { Typography } from '@mui/material'
const PillarTableHead: React.FC = (): JSX.Element => {
  return (
    <TableHead>
      <TableRow>
        <TableCell
          key="Pillar title"
          id="pillarTitle"
          sx={{ border: 1, backgroundColor: '#DCDCDC' }}
          align="center"
          size="small"
        >
          <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
            Pillars
          </Typography>
        </TableCell>
        <TableCell
          key="Description"
          id="description"
          sx={{ border: 1, backgroundColor: '#DCDCDC' }}
          align="center"
          size="small"
        >
          <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
            Score
          </Typography>
        </TableCell>
        <TableCell
          key="Notes"
          id="notes"
          sx={{ border: 1, backgroundColor: '#DCDCDC' }}
          align="center"
          size="small"
        >
          {' '}
          <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
            Notes
          </Typography>{' '}
        </TableCell>
      </TableRow>
    </TableHead>
  )
}

export default PillarTableHead
