import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';

function Sparkline({ data = [] }) {
  if (!data.length) return <span>-</span>;
  const max = Math.max(...data, 1);
  const points = data
    .map((v, i) => `${(i / Math.max(data.length - 1, 1)) * 100},${30 - (v / max) * 28}`)
    .join(' ');
  return (
    <svg width="120" height="30" viewBox="0 0 100 30">
      <polyline fill="none" stroke="#2563eb" strokeWidth="2" points={points} />
    </svg>
  );
}

function Badge({ up }) {
  return <span style={{ padding: '2px 8px', borderRadius: 12, color: '#fff', background: up ? '#16a34a' : '#dc2626' }}>{up ? 'UP' : 'DOWN'}</span>;
}

function App() {
  const [items, setItems] = useState([]);
  const [history, setHistory] = useState({});
  const [selected, setSelected] = useState('');

  async function loadStatus() {
    const res = await fetch('/api/status');
    if (!res.ok) return;
    const data = await res.json();
    setItems(data.items || []);
    if (!selected && data.items?.length) setSelected(data.items[0].name);
  }

  async function loadHistory(name) {
    if (!name) return;
    const res = await fetch(`/api/history?name=${encodeURIComponent(name)}`);
    if (!res.ok) return;
    const data = await res.json();
    setHistory((prev) => ({ ...prev, [name]: data.items || [] }));
  }

  useEffect(() => {
    loadStatus();
    const t = setInterval(loadStatus, 5000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    items.forEach((s) => loadHistory(s.name));
  }, [items.length]);

  const summary = useMemo(() => {
    const up = items.filter((x) => x.up).length;
    return { total: items.length, up, down: items.length - up };
  }, [items]);

  const selectedHistory = history[selected] || [];

  return (
    <div style={{ fontFamily: 'Inter,system-ui,sans-serif', background: '#f8fafc', minHeight: '100vh', padding: 24 }}>
      <h1 style={{ marginTop: 0 }}>test1 - Server Status Probe</h1>

      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        <Card title="总节点" value={summary.total} />
        <Card title="在线" value={summary.up} color="#16a34a" />
        <Card title="离线" value={summary.down} color="#dc2626" />
      </div>

      <div style={{ background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0', overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead style={{ background: '#f1f5f9' }}>
            <tr>
              <Th>节点</Th><Th>类型</Th><Th>目标</Th><Th>状态</Th><Th>延迟</Th><Th>24h 可用率</Th><Th>趋势</Th><Th>更新时间</Th>
            </tr>
          </thead>
          <tbody>
            {items.map((s) => {
              const h = (history[s.name] || []).slice(0, 20).reverse().map((x) => x.latencyMs || 0);
              return (
                <tr key={s.name} onClick={() => setSelected(s.name)} style={{ cursor: 'pointer', background: selected === s.name ? '#eef2ff' : '#fff' }}>
                  <Td>{s.name}</Td>
                  <Td>{s.type}</Td>
                  <Td>{s.addr}</Td>
                  <Td><Badge up={s.up} /></Td>
                  <Td>{s.latencyMs} ms</Td>
                  <Td>{s.uptime24h?.toFixed(2)}%</Td>
                  <Td><Sparkline data={h} /></Td>
                  <Td>{new Date(s.checkedAt).toLocaleString()}</Td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div style={{ marginTop: 16, background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0', padding: 16 }}>
        <h3 style={{ marginTop: 0 }}>节点历史：{selected || '-'}</h3>
        <div style={{ maxHeight: 280, overflow: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead><tr><Th>时间</Th><Th>状态</Th><Th>延迟</Th><Th>消息</Th></tr></thead>
            <tbody>
              {selectedHistory.map((r) => (
                <tr key={r.id}><Td>{new Date(r.checkedAt).toLocaleString()}</Td><Td>{r.up ? 'UP' : 'DOWN'}</Td><Td>{r.latencyMs} ms</Td><Td>{r.message}</Td></tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function Card({ title, value, color = '#111827' }) {
  return <div style={{ background: '#fff', border: '1px solid #e2e8f0', borderRadius: 12, padding: 16, minWidth: 140 }}><div style={{ color: '#64748b', fontSize: 13 }}>{title}</div><div style={{ fontSize: 26, fontWeight: 700, color }}>{value}</div></div>;
}
function Th({ children }) { return <th style={{ textAlign: 'left', padding: 10, fontSize: 13, color: '#334155' }}>{children}</th>; }
function Td({ children }) { return <td style={{ padding: 10, borderTop: '1px solid #f1f5f9', fontSize: 14 }}>{children}</td>; }

createRoot(document.getElementById('root')).render(<App />);
