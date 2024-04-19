import React from 'react';
import { BrowserRouter, Routes, Route} from 'react-router-dom';
import Home from './pages/Home'
import SharedLayout from './pages/SharedLayout';
import Pillars from './pages/Pillars'
import Identity from './pages/Identity';
import Devices from './pages/Devices';
import Network from './pages/Network';
import Application from './pages/Application';
import Data from './pages/Data';
import "./App.css";
import SurveyComponent from './components/Survey';
import SurveyPage from './pages/SurveyPage';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<SharedLayout />}>
          <Route index element={<Home />}></Route>
          <Route path="/Pillars/:fismaSystem" element={<Pillars />}></Route>
          <Route path="/Pillars/:system/Identity" element={<Identity />}/>
          <Route path="/Pillars/:system/Devices" element={<Devices />}></Route>
          <Route path="/Pillars/:system/Network" element={<Network />}></Route>
          <Route path="/Pillars/:system/Application" element={<Application />}></Route>
          <Route path="/Pillars/:system/Data" element={<Data />}></Route>
          <Route path="Questionnare" element={<SurveyPage/>}></Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}


export default App;
