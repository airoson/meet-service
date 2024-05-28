import { HashRouter, Route, Routes } from 'react-router-dom';
import './App.css';
import Header from './Components/Header';
import Login from './Components/Login';
import Signup from './Components/Signup';
import Room from './Components/Room';
import React, {useState} from 'react'
import User from './Components/User';
import Home from './Components/Home';

export const UserContext = React.createContext({ accessToken: null })

function App() {
  const [user, setUser] = useState({
    accessToken: window.localStorage.getItem("accessToken"),
    identifier: window.localStorage.getItem("identifier")})
  return (
    <div className="App">
      <UserContext.Provider value={{user, setUser}}>
        <Header />
        <HashRouter>
          <Routes>
            <Route exact path="/login" element={<Login />} />
            <Route exact path="/signup" element={<Signup />} />
            <Route exact path="/room/:roomId" element={<Room />} />
            <Route exact path="/user" element={<User />} />
            <Route exact path="/" element={<Home/>} />
          </Routes>
        </HashRouter>
      </UserContext.Provider>
    </div>
  );
}

export default App;
