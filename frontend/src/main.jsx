import React, { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';

const API = '/api/status';

function App() {
  const [items, setItems] = useState([]);

  async function load() {
    const res = await fetch(API);
    if (!res.ok) return;
    const data = await res.json();
    setItems(data.items || []);
  }

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, []);

  return (
    <div style={{ fontFamily: 'sans-serif', padding: 20 }}>
      <h2>Server Probe (test1)</h2>
      <table border="1" cellPadding="8" style={{ borderCollapse: 'collapse', width: '100%' }}>
        <thead><tr><th>Name</th><th>Type</th><th>Target</th><th>Status</th><th>Latency</th><th>Message</th><th>Checked</th></tr></thead>
        <tbody>
          {items.map((s) => (
            <tr key={s.name} style={{ background: s.up ? '#eaffea' : '#ffecec' }}>
              <td>{s.name}</td><td>{s.type}</td><td>{s.addr}</td><td>{s.up ? 'UP' : 'DOWN'}</td><td>{s.latencyMs} ms</td><td>{s.message}</td><td>{new Date(s.checkedAt).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
