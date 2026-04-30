'use client';

import { useState } from 'react';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';

const TABS = [
  { key: 'quick', label: '快速参考' },
  { key: 'path', label: '路径语法' },
  { key: 'tester', label: '在线测试' },
  { key: 'zabbix', label: 'Zabbix' },
  { key: 'prometheus', label: 'Prometheus' },
  { key: 'generic', label: '通用 JSON' },
] as const;

type TabKey = (typeof TABS)[number]['key'];

const FIELD_MAP = [
  { field: '标题', paths: 'title / alertname', note: '告警标题' },
  { field: '描述', paths: 'description / message', note: '告警详情' },
  { field: '严重等级', paths: 'severity', note: 'critical / warning / info' },
  { field: '源 IP', paths: 'source_ip', note: '告警源 IP 地址' },
  { field: '设备名称', paths: 'device_name', note: '告警设备名' },
];

const ZABBIX_MAP = [
  { field: '标题', path: 'subject', fallback: 'event.name' },
  { field: '描述', path: 'message', fallback: '-' },
  { field: '严重等级', path: 'event.severity', fallback: '-' },
  { field: '源 IP', path: 'host.ip', fallback: '-' },
  { field: '设备名称', path: 'host.name', fallback: '-' },
];

const PROMETHEUS_MAP = [
  { field: '标题', path: 'alerts.0.labels.alertname' },
  { field: '描述', path: 'alerts.0.annotations.summary', fallback: 'alerts.0.annotations.description' },
  { field: '严重等级', path: 'alerts.0.labels.severity' },
  { field: '源 IP', path: 'alerts.0.labels.instance' },
  { field: '设备名称', path: 'alerts.0.labels.device' },
  { field: '触发时间', path: 'alerts.0.startsAt (RFC3339)' },
];

const SEVERITY_TABLE = [
  { input: 'critical / crit / p1 / emerg', output: 'critical' },
  { input: 'warning / warn / p2 / high', output: 'warning' },
  { input: '其他值', output: 'info' },
];

const PATH_EXAMPLES = [
  {
    syntax: '点号访问',
    desc: '用 . 连接嵌套层级',
    expr: 'host.name',
    data: '{"host":{"name":"server-01","ip":"10.0.0.1"}}',
    result: '"server-01"',
  },
  {
    syntax: '数组索引',
    desc: '用数字索引访问数组元素（从 0 开始）',
    expr: 'alerts.0.labels.alertname',
    data: '{"alerts":[{"labels":{"alertname":"CPUHigh"}}]}',
    result: '"CPUHigh"',
  },
  {
    syntax: '键名含点号',
    desc: '用反斜杠转义含 . 的键名',
    expr: 'event\\.name',
    data: '{"event.name":"CPU告警","severity":"critical"}',
    result: '"CPU告警"',
  },
  {
    syntax: '多层嵌套',
    desc: '支持任意深度的嵌套路径',
    expr: 'data.server.metrics.cpu',
    data: '{"data":{"server":{"metrics":{"cpu":95,"mem":80}}}}',
    result: '95',
  },
];

const TESTER_PRESETS = [
  {
    label: 'Zabbix 告警',
    data: `{
  "subject": "CPU 使用率过高 on server-01",
  "message": "CPU 使用率: 95%",
  "event.severity": "high",
  "host": {
    "name": "server-01",
    "ip": "192.168.1.100"
  }
}`,
    expr: 'host.name',
  },
  {
    label: 'Prometheus 告警',
    data: `{
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "CPUHigh",
        "severity": "critical",
        "instance": "10.0.0.1"
      },
      "annotations": {
        "summary": "CPU 使用率超过 90%"
      },
      "startsAt": "2026-04-29T10:00:00Z"
    }
  ]
}`,
    expr: 'alerts.0.labels.alertname',
  },
  {
    label: '通用 JSON',
    data: `{
  "title": "链路中断",
  "description": "核心交换机端口 down",
  "severity": "critical",
  "source_ip": "10.0.0.254",
  "device_name": "core-switch-01"
}`,
    expr: 'title',
  },
];

function CodeBlock({ children }: { children: string }) {
  return (
    <pre className="overflow-x-auto rounded-md bg-gray-900 p-3 text-xs leading-relaxed text-gray-100">
      <code>{children}</code>
    </pre>
  );
}

/**
 * Minimal GJSON-like path resolver.
 * Supports: dot notation, array index, escaped dots (\.).
 */
function resolvePath(json: string, path: string): { value: string; error: string | null } {
  if (!path.trim()) return { value: '', error: null };

  let obj: unknown;
  try {
    obj = JSON.parse(json);
  } catch {
    return { value: '', error: 'JSON 格式无效' };
  }

  // Split path by unescaped dots.
  const keys = path.split(/(?<!\\)\./).map((k) => k.replace(/\\./g, '.'));

  let current: unknown = obj;
  for (const key of keys) {
    if (current == null || typeof current !== 'object') {
      return { value: '', error: null };
    }
    if (Array.isArray(current)) {
      const idx = Number(key);
      if (Number.isNaN(idx)) return { value: '', error: null };
      current = current[idx];
    } else {
      current = (current as Record<string, unknown>)[key];
    }
  }

  if (current === undefined) return { value: '', error: null };
  return { value: JSON.stringify(current, null, 2), error: null };
}

export default function HelpPage() {
  const [tab, setTab] = useState<TabKey>('quick');

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <h2 className="mb-4 text-xl font-bold text-gray-800">解析器配置指引</h2>

          <div className="mb-6 flex flex-wrap gap-1 rounded-lg bg-gray-100 p-1">
            {TABS.map((t) => (
              <button
                key={t.key}
                onClick={() => setTab(t.key)}
                className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                  tab === t.key
                    ? 'bg-white text-blue-600 shadow-sm'
                    : 'text-gray-600 hover:text-gray-800'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>

          <div className="max-w-3xl space-y-6">
            {tab === 'quick' && <QuickRef />}
            {tab === 'path' && <PathGuide />}
            {tab === 'tester' && <PathTester />}
            {tab === 'zabbix' && <ZabbixGuide />}
            {tab === 'prometheus' && <PrometheusGuide />}
            {tab === 'generic' && <GenericGuide />}
          </div>
        </main>
      </div>
    </div>
  );
}

function QuickRef() {
  return (
    <>
      <Section title="什么是解析器配置？">
        <p className="text-sm text-gray-600">
          解析器配置（JSON）用于告诉平台如何从告警数据中提取关键信息。
          <strong> Zabbix 和 Prometheus 类型已内置解析规则，通常填 <code className="rounded bg-gray-100 px-1">{}</code> 即可。</strong>
          只有"通用"类型需要自定义配置。
        </p>
      </Section>

      <Section title="平台提取的字段">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="py-2 text-left font-medium text-gray-600">字段</th>
              <th className="py-2 text-left font-medium text-gray-600">JSON 路径</th>
              <th className="py-2 text-left font-medium text-gray-600">说明</th>
            </tr>
          </thead>
          <tbody>
            {FIELD_MAP.map((row) => (
              <tr key={row.field} className="border-b border-gray-100">
                <td className="py-2 font-medium">{row.field}</td>
                <td className="py-2 font-mono text-xs text-gray-600">{row.paths}</td>
                <td className="py-2 text-gray-500">{row.note}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Section>

      <Section title="严重等级映射规则">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="py-2 text-left font-medium text-gray-600">原始值</th>
              <th className="py-2 text-left font-medium text-gray-600">映射为</th>
            </tr>
          </thead>
          <tbody>
            {SEVERITY_TABLE.map((row) => (
              <tr key={row.input} className="border-b border-gray-100">
                <td className="py-2 font-mono text-xs text-gray-600">{row.input}</td>
                <td className="py-2">
                  <SeverityBadge level={row.output} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Section>
    </>
  );
}

function PathGuide() {
  return (
    <>
      <Section title="路径语法 (GJSON Path)">
        <p className="mb-3 text-sm text-gray-600">
          平台使用 <strong>GJSON 路径语法</strong>从 JSON 数据中提取字段值。语法类似 JSONPath，但更简洁。
        </p>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="py-2 text-left font-medium text-gray-600">语法</th>
              <th className="py-2 text-left font-medium text-gray-600">说明</th>
              <th className="py-2 text-left font-medium text-gray-600">示例路径</th>
              <th className="py-2 text-left font-medium text-gray-600">结果</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">点号 <code className="rounded bg-gray-100 px-1">.</code></td>
              <td className="py-2 text-gray-600">访问对象的键</td>
              <td className="py-2 font-mono text-xs text-blue-600">host.name</td>
              <td className="py-2 font-mono text-xs text-gray-600">server-01</td>
            </tr>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">数字索引</td>
              <td className="py-2 text-gray-600">访问数组元素（从 0 开始）</td>
              <td className="py-2 font-mono text-xs text-blue-600">alerts.0.labels.alertname</td>
              <td className="py-2 font-mono text-xs text-gray-600">CPUHigh</td>
            </tr>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">转义点号 <code className="rounded bg-gray-100 px-1">\.</code></td>
              <td className="py-2 text-gray-600">键名本身含点号时用反斜杠转义</td>
              <td className="py-2 font-mono text-xs text-blue-600">event\.name</td>
              <td className="py-2 font-mono text-xs text-gray-600">CPU告警</td>
            </tr>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">多层嵌套</td>
              <td className="py-2 text-gray-600">支持任意深度</td>
              <td className="py-2 font-mono text-xs text-blue-600">data.server.metrics.cpu</td>
              <td className="py-2 font-mono text-xs text-gray-600">95</td>
            </tr>
          </tbody>
        </table>
      </Section>

      <Section title="完整示例">
        <div className="space-y-4">
          {PATH_EXAMPLES.map((ex) => (
            <div key={ex.expr} className="rounded-md border border-gray-100 p-3">
              <div className="mb-1 flex items-center gap-2">
                <span className="text-sm font-medium text-gray-800">{ex.syntax}</span>
                <span className="text-xs text-gray-400">— {ex.desc}</span>
              </div>
              <div className="grid grid-cols-2 gap-3 text-xs">
                <div>
                  <div className="mb-1 font-medium text-gray-500">输入数据</div>
                  <pre className="overflow-x-auto rounded bg-gray-50 p-2 text-gray-700">{ex.data}</pre>
                </div>
                <div>
                  <div className="mb-1 font-medium text-gray-500">
                    路径: <code className="text-blue-600">{ex.expr}</code>
                  </div>
                  <pre className="overflow-x-auto rounded bg-green-50 p-2 text-green-800">{ex.result}</pre>
                </div>
              </div>
            </div>
          ))}
        </div>
      </Section>

      <Section title="路径不存在时">
        <p className="text-sm text-gray-600">
          如果路径指向的键不存在，平台会返回空值，不会报错。解析器会继续尝试备选路径（如 <code className="rounded bg-gray-100 px-1">title</code> 不存在则尝试 <code className="rounded bg-gray-100 px-1">alertname</code>）。
        </p>
      </Section>
    </>
  );
}

function PathTester() {
  const [jsonInput, setJsonInput] = useState(TESTER_PRESETS[0].data);
  const [pathExpr, setPathExpr] = useState(TESTER_PRESETS[0].expr);

  const { value, error } = resolvePath(jsonInput, pathExpr);
  const jsonValid = (() => { try { JSON.parse(jsonInput); return true; } catch { return false; } })();

  return (
    <>
      <Section title="路径测试器">
        <p className="mb-4 text-sm text-gray-600">
          输入 JSON 数据和路径表达式，实时查看匹配结果。下方有预设示例可一键填入。
        </p>

        <div className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">
              JSON 数据
              {!jsonValid && jsonInput.trim() && (
                <span className="ml-2 text-xs text-red-500">JSON 格式无效</span>
              )}
            </label>
            <textarea
              value={jsonInput}
              onChange={(e) => setJsonInput(e.target.value)}
              rows={10}
              className="w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">路径表达式</label>
            <input
              value={pathExpr}
              onChange={(e) => setPathExpr(e.target.value)}
              placeholder="例如: host.name 或 alerts.0.labels.alertname"
              className="w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div className="rounded-md border border-gray-200 bg-gray-50 p-3">
            <div className="mb-1 text-sm font-medium text-gray-700">匹配结果</div>
            {error ? (
              <div className="text-sm text-red-500">{error}</div>
            ) : value ? (
              <pre className="overflow-x-auto rounded bg-white p-2 font-mono text-sm text-green-700">{value}</pre>
            ) : (
              <div className="text-sm text-gray-400">无匹配（路径不存在或值为 null）</div>
            )}
          </div>

          <div>
            <div className="mb-2 text-sm font-medium text-gray-700">预设示例</div>
            <div className="flex gap-2">
              {TESTER_PRESETS.map((preset) => (
                <button
                  key={preset.label}
                  onClick={() => {
                    setJsonInput(preset.data);
                    setPathExpr(preset.expr);
                  }}
                  className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:border-blue-400 hover:text-blue-600"
                >
                  {preset.label}
                </button>
              ))}
            </div>
          </div>
        </div>
      </Section>
    </>
  );
}

function ZabbixGuide() {
  return (
    <>
      <Section title="Zabbix 内置字段映射">
        <p className="mb-3 text-sm text-gray-600">平台自动从 Zabbix Webhook 数据中提取以下字段，无需配置解析器。</p>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="py-2 text-left font-medium text-gray-600">提取字段</th>
              <th className="py-2 text-left font-medium text-gray-600">取值路径</th>
              <th className="py-2 text-left font-medium text-gray-600">备选路径</th>
            </tr>
          </thead>
          <tbody>
            {ZABBIX_MAP.map((row) => (
              <tr key={row.field} className="border-b border-gray-100">
                <td className="py-2 font-medium">{row.field}</td>
                <td className="py-2 font-mono text-xs text-gray-600">{row.path}</td>
                <td className="py-2 font-mono text-xs text-gray-400">{row.fallback}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Section>

      <Section title="Zabbix 端配置步骤">
        <ol className="list-inside list-decimal space-y-2 text-sm text-gray-600">
          <li>
            在 Zabbix 管理端进入 <strong>Administration → Media types → Create media type</strong>
          </li>
          <li>
            类型选择 <strong>Webhook</strong>，URL 填入：
            <CodeBlock>{'POST http://你的服务器地址/api/v1/alerts/webhook/{source_id}'}</CodeBlock>
          </li>
          <li>
            如配置了 Webhook Secret，添加 HTTP Header：
            <CodeBlock>{'X-Webhook-Secret: 你的密钥'}</CodeBlock>
          </li>
          <li>
            进入 <strong>Configuration → Actions</strong>，创建 Action 并绑定此 Media Type
          </li>
        </ol>
      </Section>

      <Section title="解析器配置">
        <p className="mb-2 text-sm text-gray-600">Zabbix 类型通常无需自定义，直接填：</p>
        <CodeBlock>{'{}'}</CodeBlock>
      </Section>
    </>
  );
}

function PrometheusGuide() {
  return (
    <>
      <Section title="Prometheus 内置字段映射">
        <p className="mb-3 text-sm text-gray-600">平台自动从 Alertmanager Webhook 数据中提取以下字段。</p>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="py-2 text-left font-medium text-gray-600">提取字段</th>
              <th className="py-2 text-left font-medium text-gray-600">取值路径</th>
            </tr>
          </thead>
          <tbody>
            {PROMETHEUS_MAP.map((row) => (
              <tr key={row.field} className="border-b border-gray-100">
                <td className="py-2 font-medium">{row.field}</td>
                <td className="py-2 font-mono text-xs text-gray-600">{row.path}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Section>

      <Section title="Alertmanager 配置">
        <p className="mb-2 text-sm text-gray-600">在 alertmanager.yml 中添加：</p>
        <CodeBlock>{`receivers:
  - name: 'network-ticket'
    webhook_configs:
      - url: 'http://你的服务器/api/v1/alerts/webhook/{source_id}'
        send_resolved: true`}</CodeBlock>
      </Section>

      <Section title="解析器配置">
        <p className="mb-2 text-sm text-gray-600">Prometheus 类型通常无需自定义，直接填：</p>
        <CodeBlock>{'{}'}</CodeBlock>
      </Section>
    </>
  );
}

function GenericGuide() {
  return (
    <>
      <Section title="通用 JSON 格式说明">
        <p className="text-sm text-gray-600">
          适用于自定义监控系统。平台从 JSON 请求体中按字段名提取数据，
          <strong>支持嵌套路径</strong>（如 <code className="rounded bg-gray-100 px-1">data.host.ip</code>）。
        </p>
      </Section>

      <Section title="最小可用示例">
        <p className="mb-2 text-sm text-gray-600">只需要发送如下 JSON 即可创建工单：</p>
        <CodeBlock>{`{
  "title": "CPU 使用率过高",
  "description": "server-01 CPU 超过 90%",
  "severity": "critical",
  "source_ip": "192.168.1.100",
  "device_name": "server-01"
}`}</CodeBlock>
      </Section>

      <Section title="字段说明">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200">
              <th className="py-2 text-left font-medium text-gray-600">字段</th>
              <th className="py-2 text-left font-medium text-gray-600">JSON 键名</th>
              <th className="py-2 text-left font-medium text-gray-600">必填</th>
              <th className="py-2 text-left font-medium text-gray-600">说明</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">标题</td>
              <td className="py-2 font-mono text-xs text-gray-600">title / alertname</td>
              <td className="py-2 text-red-500">是</td>
              <td className="py-2 text-gray-500">两个键名任选一个</td>
            </tr>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">描述</td>
              <td className="py-2 font-mono text-xs text-gray-600">description / message</td>
              <td className="py-2">否</td>
              <td className="py-2 text-gray-500">两个键名任选一个</td>
            </tr>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">严重等级</td>
              <td className="py-2 font-mono text-xs text-gray-600">severity</td>
              <td className="py-2">否</td>
              <td className="py-2 text-gray-500">critical / warning / info</td>
            </tr>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">源 IP</td>
              <td className="py-2 font-mono text-xs text-gray-600">source_ip</td>
              <td className="py-2">否</td>
              <td className="py-2 text-gray-500">告警源 IP 地址</td>
            </tr>
            <tr className="border-b border-gray-100">
              <td className="py-2 font-medium">设备名称</td>
              <td className="py-2 font-mono text-xs text-gray-600">device_name</td>
              <td className="py-2">否</td>
              <td className="py-2 text-gray-500">告警设备名</td>
            </tr>
          </tbody>
        </table>
      </Section>

      <Section title="解析器配置">
        <p className="mb-2 text-sm text-gray-600">通用类型直接填：</p>
        <CodeBlock>{'{}'}</CodeBlock>
        <p className="mt-2 text-xs text-gray-400">平台会自动从 JSON 中按上述键名提取字段。</p>
      </Section>
    </>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-5">
      <h3 className="mb-3 text-base font-semibold text-gray-800">{title}</h3>
      {children}
    </div>
  );
}

function SeverityBadge({ level }: { level: string }) {
  const styles: Record<string, string> = {
    critical: 'bg-red-100 text-red-800',
    warning: 'bg-yellow-100 text-yellow-800',
    info: 'bg-blue-100 text-blue-800',
  };
  return (
    <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${styles[level] || 'bg-gray-100 text-gray-800'}`}>
      {level}
    </span>
  );
}
