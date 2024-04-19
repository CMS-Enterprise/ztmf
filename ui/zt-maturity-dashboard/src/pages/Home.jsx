import React, { useState, useEffect } from "react";
import Dropdown from "../components/Dropdown";
import Select from "react-select";
import NextButton from "../components/NextButton";
import FISMASystems from "../FISMA Systems";
import { Outlet } from "react-router-dom";
import { useQuery, gql } from "@apollo/client";
import { Button } from "@cmsgov/design-system";
import { Link } from "react-router-dom";

export default function Home() {
    const [isButton, setIsButton] = useState("");
    const [fismaSystem, setFismaSystem] = useState("");
    const [fismaSystemSet, setFismaSystemSet] = useState(new Set());
    const [fismaSystemList, setFismaSystemList] = useState(null);
    const [options, setOptions] = useState([
      {'value': 0, 'label': "- Select a system -" },
    ]);
    const handleChange = (selectedOption) => {
      setIsButton(selectedOption.value);
      setFismaSystem(selectedOption.label)
    };
    const FISMASYSTEMS_QUERY = gql`
        query getFismaSystems {
          fismasystems {
            fismasystemid
            fismaacronym
          }
        }
      `;
    const { data, loading, error } = useQuery(FISMASYSTEMS_QUERY);
    useEffect(() => {
      if (!loading && !error && data) {
        setFismaSystemList(data.fismasystems)
        data.fismasystems.map((system) => {
          const row = [
            { 'value': Number(system.fismasystemid), 'label': system.fismaacronym },
          ];
          setOptions(options => [...options,...row])
          const updatedSet = new Set(fismaSystemSet);
          updatedSet.add(system.fismaacronym);
          setFismaSystemSet(updatedSet);
        })
        console.log(Array.from(fismaSystemSet))
      }
    }, [loading, error, data]);
    if (loading) return "Loading...";
    if (error) return <pre>{error.message}</pre>;
    return (
      <div>
        <div className="ds-l-row">
          <div className="ds-l-md-col--6 ds-l-sm-col--12">
            <h1 className="ds-u-md-text-align--left ds-u-margin-bottom--7">
              Welcome to the Zero Trust Maturity score dashboard! This dashboard
              attempts to breakdown data silos and...
            </h1>
            <div>
              <Select
                defaultValue={options[0]}
                isSearchable={true}
                options={options}
                components={{ Dropdown }}
                onChange={handleChange}
              />
            </div>
            <div className="ds-u-display--flex ds-u-margin-top--7">
              {isButton > 0 && <NextButton linkTo={`Pillars/${fismaSystem}`} />}
            </div>
          </div>
          {/* <div className="ds-l-md-col--6 ds-l-sm-col--12">
            <div className="ds-u-display--flex  ds-u-justify-content--end ds-u-margin-top--3">
              <div className="ds-u-justify-content--end">
                <Link to={`Questionnare`}>
                  <Button
                    className='ds-u-margin-right--1"'
                    onAnalyticsEvent={function noRefCheck() {}}>
                    Survey Page
                  </Button>
                </Link>
              </div>
            </div>
          </div> */}
        </div>
        <Outlet />
      </div>
    );
    

}