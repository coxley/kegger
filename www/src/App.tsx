import React, { useEffect, useReducer, useState } from 'react';

import useWebSocket from 'react-use-websocket';


function App() {
  const stats = useStats()
  const latestPour = usePours()
  return (
    <div className="h-screen w-screen bg-blue-200">
      <div className="grid grid-cols-6 grid-rows-6 h-screen">
        {Object.entries(stats.records).map((item) => {
          const name = item[0]
          const records = item[1]

          let total_ounces = 0
          let total_pulses = 0
          for (let rec in records) {
            total_ounces += records[rec].ounces
            total_pulses += records[rec].pulses
          }
          let classes = ""
          if (name === "1") {
            classes = "col-start-2 col-end-4"
          } else if (name === "2") {
            classes = "col-start-4 col-end-6"
          }


          const pourStatus = (latestPour.tap.toString() === name && latestPour.active) ? <span className="text-green-500">{latestPour.ounces.toFixed(2)}oz</span> : <span className="text-orange-500">Inactive</span>
          return (
            <div className={"row-start-2 row-end-5 bg-gray-100 border-gray-800 rounded-md shadow-md h-auto m-10" + " " + classes}>
              <div className="flex items-end h-12 border-b-gray-300 shadow-sm border-b bg-gray-50">
                <span className="pb-2 pl-2">
                  Tap {name}
                </span>
              </div>
              <div className="isolate flex flex-col px-2 pt-4">
                <div className="flex flex-row justify-between">
                  <span className="text-left font-medium">Pour Status</span>
                  <span className="text-right">{pourStatus}</span>
                </div>
                <div className="flex flex-row justify-between">
                  <span className="text-left font-medium">Total Pours</span>
                  <span className="text-right">{records.length}</span>
                </div>
                <div className="flex flex-row justify-between">
                  <span className="text-left font-medium">Total Ounces</span>
                  <span className="text-right">{total_ounces.toFixed(2)}oz</span>
                </div>
                <div className="flex flex-row justify-between">
                  <span className="text-left font-medium">Total Pulses</span>
                  <span className="text-right">{total_pulses}</span>
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div >
  );
}

interface Stats {
  total_pulses: number,
  total_ounces: number,
  records: { [tap: number]: Record[] },
}

interface Record {
  timestamp: number,
  tap: number,
  pulses: number,
  ounces: number,
}

const addr = "ws://192.168.100.126:80/stats"
function useStats(): Stats {
  const { lastJsonMessage } = useWebSocket(addr, {
    shouldReconnect: (_) => true,
  });

  if (!lastJsonMessage) {
    return { total_pulses: 0, total_ounces: 0, records: [] }
  }
  return lastJsonMessage as unknown as Stats
}

const pourAddr = "ws://192.168.100.126:80/pours"
type pourState = {
  timestamp: number, tap: number, active: boolean, pulses: number, ounces: number
}
function usePours(): pourState {
  const { lastJsonMessage } = useWebSocket(pourAddr, {
    shouldReconnect: (_) => true,
  });

  if (!lastJsonMessage) {
    return { timestamp: 0, tap: 0, active: false, pulses: 0, ounces: 0 }
  }

  // TODO: adjust to allow both to be active
  // TODO: flow rate, etc
  return lastJsonMessage as unknown as pourState
}



export default App;
