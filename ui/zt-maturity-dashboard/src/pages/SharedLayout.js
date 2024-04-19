import React from "react";
// import Head from "../components/Header";
import Title from "../components/Title";
import {Outlet} from "react-router-dom"
export default function SharedLayout() {
    return (
      <div className="App">
        {/* <Head /> */}
        <main className="ds-base ds-l-container example-grid">
            <Title/>
            <Outlet />
        </main>
      </div>
    );
}