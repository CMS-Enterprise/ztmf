import * as React from 'react'
import Typography from '@mui/material/Typography'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import FormControl from '@mui/material/FormControl'
import NavigateNextIcon from '@mui/icons-material/NavigateNext'
import Select, { SelectChangeEvent } from '@mui/material/Select'
import { useQuery } from '@apollo/client'
import Button from '@mui/material/Button'
import gql from 'graphql-tag'
import { Link } from 'react-router-dom'
import { DocumentNode } from '@apollo/client'

/**
 * Component that renders the contents of the Dashboard view.
 * @returns {JSX.Element} Component that renders the dashboard contents.
 */

const FISMASYSTEMS_QUERY: DocumentNode = gql(`
    query getFismaSystems {
    fismasystems {
        fismasystemid
        fismaacronym
        fismasubsystem
    }}
`)
type SYSTEMPROPS = {
  id: number[]
  subsystem: string[]
}
type QUERYPROPS = {
  fismasystemid: number
  fismaacronym: string
  fismasubsystem: string
}
type FISMASYSTEM = {
  fismaacronym: string
  fismasubsystem: string
  fismasystemid: number
}
type Dictionary = { [key: string]: SYSTEMPROPS }

const HomePageContainer: React.FC = (): JSX.Element => {
  const [fismaSystem, setFismaSystem] = React.useState<string>('')
  const [fismaSubsystem, setFimsaSubsystem] = React.useState<string>('')
  const [fismaTable, setFismaTable] = React.useState<Dictionary>({})
  const [haveSubsystem, setHaveSubsystem] = React.useState<boolean>(false)
  const [subsystem, setSubsystem] = React.useState<string[]>([])
  const [subsystemId, setSubsystemId] = React.useState<number[]>([])
  const [fismaSystemID, setID] = React.useState(-1)
  const { loading, data } = useQuery(FISMASYSTEMS_QUERY)
  const handleChange = (event: SelectChangeEvent) => {
    setFismaSystem(event.target.value as string)
  }
  const handleSubsystemChange = (event: SelectChangeEvent) => {
    setFimsaSubsystem(event.target.value as string)
    setID(Number(event.target.value))
  }
  const handleSubsystemClick = (event: React.MouseEvent<HTMLLIElement>) => {
    const subsystem = (event.target as HTMLElement).innerText.trimEnd()
    // console.log(event.target.value)
    setFimsaSubsystem(subsystem)
  }

  const handleClick = (event: React.MouseEvent<HTMLLIElement>) => {
    const acronym = (event.target as HTMLElement).innerText.trimEnd()
    const fismasubsystem = fismaTable[acronym].subsystem
    const fismaSubsystemId = fismaTable[acronym].id
    // console.log(subsystem)
    if (fismasubsystem.length > 1) {
      setHaveSubsystem(true)
      setSubsystem(fismasubsystem)
      setSubsystemId(fismaSubsystemId)
      setID(-1)
    } else {
      setHaveSubsystem(false)
      setSubsystemId([])
      setSubsystem([''])
      setID(fismaSubsystemId[0])
    }
  }

  // TODO: refactor this code to use a loader
  React.useEffect(() => {
    if (data) {
      const seenAcronyms: Set<string> = new Set()
      const newDictionary: Dictionary = {}
      data.fismasystems.forEach((system: FISMASYSTEM) => {
        if (!seenAcronyms.has(system.fismaacronym)) {
          seenAcronyms.add(system.fismaacronym)
          newDictionary[system.fismaacronym] = {
            id: [system.fismasystemid],
            subsystem: [system.fismasubsystem],
          }
        } else {
          const existingSystem = newDictionary[system.fismaacronym]
          // Push the new id and subsystem into the existing SYSTEMPROPS object
          existingSystem.id.push(system.fismasystemid)
          existingSystem.subsystem.push(system.fismasubsystem)
        }
      })

      setFismaTable((prevDictionary) => ({
        ...prevDictionary,
        ...newDictionary,
      }))
    }
  }, [data])
  return (
    <>
      {loading ? (
        <p>Loading ...</p>
      ) : (
        <>
          <div>
            <Typography variant="h6" sx={{ my: 2 }} align="left">
              Welcome to the Zero Trust Maturity score dashboard!
              <br />
              This dashboard attempts to breakdown data silos and...
            </Typography>
            <FormControl sx={{ m: 1, width: 400 }}>
              <InputLabel id="fisma-select-label" sx={{ marginTop: 0 }}>
                FISMA Systems
              </InputLabel>
              <Select
                labelId="fisma-select-label"
                id="fisma-select"
                value={fismaSystem}
                label="FISMA Systems"
                onChange={handleChange}
                MenuProps={{ PaperProps: { sx: { maxHeight: 300 } } }}
              >
                {data &&
                  (() => {
                    const seenAcronyms: Set<string> = new Set()
                    return data.fismasystems.map(
                      (system: QUERYPROPS, index: number) => {
                        if (!seenAcronyms.has(system.fismaacronym)) {
                          seenAcronyms.add(system.fismaacronym)
                          return (
                            <MenuItem
                              key={index}
                              value={system.fismasystemid}
                              onClick={handleClick}
                            >
                              {system.fismaacronym}
                            </MenuItem>
                          )
                        }
                      }
                    )
                  })()}
              </Select>
            </FormControl>
            {haveSubsystem && (
              <FormControl sx={{ m: 1, width: 400 }}>
                <InputLabel id="fisma-subsystem-select-label">
                  FISMA Subsystems
                </InputLabel>
                <Select
                  labelId="fisma-subsystem-select-label"
                  id="fisma-subsystem-select"
                  value={fismaSubsystem}
                  label="FISMA Subysystem"
                  onChange={handleSubsystemChange}
                  MenuProps={{ PaperProps: { sx: { maxHeight: 300 } } }}
                >
                  {subsystem &&
                    (() => {
                      return subsystem.map((ss: string, index: number) => {
                        let acro: string = ''
                        if (ss !== '') {
                          acro = ss
                        } else {
                          acro = 'None'
                        }
                        return (
                          <MenuItem
                            key={index}
                            value={subsystemId[index]}
                            onClick={handleSubsystemClick}
                          >
                            {acro}
                          </MenuItem>
                        )
                      })
                    })()}
                </Select>
              </FormControl>
            )}
            <div>
              {fismaSystemID != -1 && (
                <Link to={`/pillars/${fismaSystemID}`}>
                  <Button
                    variant="contained"
                    sx={{ mt: 10, ml: 1, marginLeft: 0, marginTop: 4 }}
                    endIcon={<NavigateNextIcon />}
                  >
                    Next
                  </Button>
                </Link>
              )}
            </div>
          </div>
        </>
      )}
    </>
  )
}
export default HomePageContainer
