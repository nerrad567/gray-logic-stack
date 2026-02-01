/**
 * KNX Technical Reference Content
 * 
 * Comprehensive guide for KNX addressing, patterns, and best practices.
 */

export const KNX_REFERENCE = {
  // ═══════════════════════════════════════════════════════════════════════════
  // GROUP ADDRESS STRUCTURE
  // ═══════════════════════════════════════════════════════════════════════════
  gaStructure: {
    title: "Group Address Structure",
    content: `
      <h4>3-Level Addressing (Main/Middle/Sub)</h4>
      <p>KNX uses a hierarchical addressing scheme: <code>Main/Middle/Sub</code></p>
      
      <table class="ref-table">
        <tr><th>Level</th><th>Range</th><th>Typical Use</th></tr>
        <tr><td>Main Group</td><td>0-31</td><td>Function type or building zone</td></tr>
        <tr><td>Middle Group</td><td>0-7</td><td>Floor/location or sub-function</td></tr>
        <tr><td>Sub Group</td><td>0-255</td><td>Individual object</td></tr>
      </table>

      <h4>Reserved Addresses</h4>
      <ul>
        <li><strong>0/x/x</strong> — Often reserved for central/system functions</li>
        <li><strong>14/x/x</strong> — Sometimes used for building services</li>
        <li><strong>15/x/x</strong> — Sometimes used for diagnostics/commissioning</li>
        <li><strong>x/x/0</strong> — Some use for "all in group" broadcast</li>
      </ul>
    `,
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // LAYOUT PATTERNS
  // ═══════════════════════════════════════════════════════════════════════════
  layoutPatterns: {
    title: "Common GA Layout Patterns",
    content: `
      <h4>Residential / Intuitive (Function-Based)</h4>
      <p>Main groups by function, middle groups by floor:</p>
      <pre class="ref-code">
1/x/x  Lighting
2/x/x  Blinds/Shutters
3/x/x  HVAC
4/x/x  Sensors
5/x/x  Scenes

1/0/x  Lighting - Ground Floor
1/1/x  Lighting - First Floor
1/2/x  Lighting - Second Floor
      </pre>

      <h4>Small/Medium Commercial (Location-Based)</h4>
      <p>Main groups by floor/zone, middle groups by function:</p>
      <pre class="ref-code">
1/x/x  Ground Floor
2/x/x  First Floor
3/x/x  Second Floor

1/0/x  Ground Floor - Lighting
1/1/x  Ground Floor - Blinds
1/2/x  Ground Floor - HVAC
      </pre>

      <h4>Large Commercial / Campus</h4>
      <p>Main groups by building/zone, more granular hierarchy:</p>
      <pre class="ref-code">
1/x/x  Building A
2/x/x  Building B
...
1/0/x  Building A - Zone 1
1/1/x  Building A - Zone 2
      </pre>
    `,
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // DEVICE GA TEMPLATES
  // ═══════════════════════════════════════════════════════════════════════════
  deviceTemplates: {
    title: "Typical GA Assignments by Device",
    content: `
      <h4>Light Switch</h4>
      <table class="ref-table">
        <tr><th>Function</th><th>DPT</th><th>Flags</th><th>Description</th></tr>
        <tr><td>switch</td><td>1.001</td><td>CWT</td><td>On/Off command</td></tr>
        <tr><td>switch_status</td><td>1.001</td><td>CRT</td><td>Current state feedback</td></tr>
      </table>

      <h4>Dimmer</h4>
      <table class="ref-table">
        <tr><th>Function</th><th>DPT</th><th>Flags</th><th>Description</th></tr>
        <tr><td>switch</td><td>1.001</td><td>CWT</td><td>On/Off command</td></tr>
        <tr><td>switch_status</td><td>1.001</td><td>CRT</td><td>On/Off feedback</td></tr>
        <tr><td>brightness</td><td>5.001</td><td>CWT</td><td>Dim level (0-100%)</td></tr>
        <tr><td>brightness_status</td><td>5.001</td><td>CRT</td><td>Current level feedback</td></tr>
      </table>

      <h4>Blind/Shutter</h4>
      <table class="ref-table">
        <tr><th>Function</th><th>DPT</th><th>Flags</th><th>Description</th></tr>
        <tr><td>move</td><td>1.008</td><td>CWT</td><td>Up/Down command</td></tr>
        <tr><td>stop</td><td>1.007</td><td>CWT</td><td>Stop movement</td></tr>
        <tr><td>position</td><td>5.001</td><td>CWT</td><td>Position (0=open, 100=closed)</td></tr>
        <tr><td>position_status</td><td>5.001</td><td>CRT</td><td>Current position</td></tr>
        <tr><td>slat</td><td>5.001</td><td>CWT</td><td>Slat angle (blinds only)</td></tr>
        <tr><td>slat_status</td><td>5.001</td><td>CRT</td><td>Current slat angle</td></tr>
      </table>

      <h4>Presence Sensor</h4>
      <table class="ref-table">
        <tr><th>Function</th><th>DPT</th><th>Flags</th><th>Description</th></tr>
        <tr><td>presence</td><td>1.018</td><td>CRT</td><td>Occupied/Unoccupied</td></tr>
        <tr><td>brightness</td><td>9.004</td><td>CRT</td><td>Lux level</td></tr>
      </table>

      <h4>Temperature Sensor</h4>
      <table class="ref-table">
        <tr><th>Function</th><th>DPT</th><th>Flags</th><th>Description</th></tr>
        <tr><td>temperature</td><td>9.001</td><td>CRT</td><td>Current temperature °C</td></tr>
      </table>

      <h4>Wall Switch / Push Button</h4>
      <table class="ref-table">
        <tr><th>Function</th><th>DPT</th><th>Flags</th><th>Description</th></tr>
        <tr><td>switch</td><td>1.001</td><td>CT</td><td>Toggle/trigger on press</td></tr>
      </table>
    `,
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // COMMUNICATION FLAGS
  // ═══════════════════════════════════════════════════════════════════════════
  flags: {
    title: "Communication Flags (CRWTUI)",
    content: `
      <p>Flags control how a group object communicates on the bus:</p>
      
      <table class="ref-table">
        <tr><th>Flag</th><th>Name</th><th>Description</th></tr>
        <tr><td><strong>C</strong></td><td>Communication</td><td>Object can communicate (required for any bus activity)</td></tr>
        <tr><td><strong>R</strong></td><td>Read</td><td>Object responds to GroupRead requests</td></tr>
        <tr><td><strong>W</strong></td><td>Write</td><td>Object accepts GroupWrite commands</td></tr>
        <tr><td><strong>T</strong></td><td>Transmit</td><td>Object sends on value change</td></tr>
        <tr><td><strong>U</strong></td><td>Update</td><td>Object updates its value from bus (for displays)</td></tr>
        <tr><td><strong>I</strong></td><td>Read on Init</td><td>Request value from bus on device startup</td></tr>
      </table>

      <h4>Common Flag Combinations</h4>
      <table class="ref-table">
        <tr><th>Flags</th><th>Use Case</th></tr>
        <tr><td>CWT</td><td>Command input (switch, button) — write to control, transmit to send</td></tr>
        <tr><td>CRT</td><td>Status output (feedback) — read to request, transmit on change</td></tr>
        <tr><td>CRWT</td><td>Bidirectional — both control and status</td></tr>
        <tr><td>CRU</td><td>Display — read and update from bus values</td></tr>
        <tr><td>CRWTUI</td><td>Full functionality — all operations enabled</td></tr>
      </table>
    `,
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // INDIVIDUAL ADDRESS
  // ═══════════════════════════════════════════════════════════════════════════
  individualAddress: {
    title: "Individual Address (IA)",
    content: `
      <h4>Format: Area.Line.Device</h4>
      
      <table class="ref-table">
        <tr><th>Component</th><th>Range</th><th>Description</th></tr>
        <tr><td>Area</td><td>0-15</td><td>Backbone/building segment</td></tr>
        <tr><td>Line</td><td>0-15</td><td>Line within area</td></tr>
        <tr><td>Device</td><td>0-255</td><td>Device on line (0 = coupler)</td></tr>
      </table>

      <h4>Reserved Addresses</h4>
      <ul>
        <li><strong>0.0.0</strong> — System address, never assign to devices</li>
        <li><strong>x.y.0</strong> — Line/area coupler (bus coupler device)</li>
        <li><strong>x.0.0</strong> — Area coupler / backbone router</li>
        <li><strong>15.15.255</strong> — Broadcast / programming mode</li>
      </ul>

      <h4>Best Practices</h4>
      <ul>
        <li>Start device numbering at 1 (0 is for couplers)</li>
        <li>Group related devices on the same line</li>
        <li>Leave gaps for future expansion (e.g., 1, 5, 10, 15...)</li>
        <li>Document your addressing scheme!</li>
      </ul>
    `,
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // COMMON DPTs
  // ═══════════════════════════════════════════════════════════════════════════
  dptReference: {
    title: "Common Data Point Types (DPT)",
    content: `
      <h4>Boolean / Switch (1 bit)</h4>
      <table class="ref-table">
        <tr><th>DPT</th><th>Name</th><th>Values</th></tr>
        <tr><td>1.001</td><td>Switch</td><td>0=Off, 1=On</td></tr>
        <tr><td>1.002</td><td>Bool</td><td>0=False, 1=True</td></tr>
        <tr><td>1.003</td><td>Enable</td><td>0=Disable, 1=Enable</td></tr>
        <tr><td>1.007</td><td>Step</td><td>0=Decrease, 1=Increase</td></tr>
        <tr><td>1.008</td><td>UpDown</td><td>0=Up, 1=Down</td></tr>
        <tr><td>1.009</td><td>OpenClose</td><td>0=Open, 1=Close</td></tr>
        <tr><td>1.018</td><td>Occupancy</td><td>0=Unoccupied, 1=Occupied</td></tr>
      </table>

      <h4>Unsigned Integer</h4>
      <table class="ref-table">
        <tr><th>DPT</th><th>Name</th><th>Range</th></tr>
        <tr><td>5.001</td><td>Percentage (0-100)</td><td>0-100%</td></tr>
        <tr><td>5.003</td><td>Angle</td><td>0-360°</td></tr>
        <tr><td>5.004</td><td>Percent (0-255)</td><td>0-255</td></tr>
        <tr><td>5.010</td><td>Counter pulses</td><td>0-255</td></tr>
        <tr><td>7.001</td><td>Pulses (16-bit)</td><td>0-65535</td></tr>
      </table>

      <h4>Float (2-byte)</h4>
      <table class="ref-table">
        <tr><th>DPT</th><th>Name</th><th>Unit</th></tr>
        <tr><td>9.001</td><td>Temperature</td><td>°C</td></tr>
        <tr><td>9.002</td><td>Temp difference</td><td>K</td></tr>
        <tr><td>9.004</td><td>Lux</td><td>lx</td></tr>
        <tr><td>9.005</td><td>Wind speed</td><td>m/s</td></tr>
        <tr><td>9.006</td><td>Pressure</td><td>Pa</td></tr>
        <tr><td>9.007</td><td>Humidity</td><td>%</td></tr>
        <tr><td>9.008</td><td>Air quality</td><td>ppm</td></tr>
        <tr><td>9.021</td><td>Current</td><td>mA</td></tr>
        <tr><td>9.024</td><td>Power</td><td>kW</td></tr>
      </table>

      <h4>Time / Date</h4>
      <table class="ref-table">
        <tr><th>DPT</th><th>Name</th><th>Format</th></tr>
        <tr><td>10.001</td><td>Time of day</td><td>HH:MM:SS</td></tr>
        <tr><td>11.001</td><td>Date</td><td>DD/MM/YYYY</td></tr>
        <tr><td>19.001</td><td>DateTime</td><td>Full timestamp</td></tr>
      </table>

      <h4>String / Text</h4>
      <table class="ref-table">
        <tr><th>DPT</th><th>Name</th><th>Max Length</th></tr>
        <tr><td>16.000</td><td>ASCII string</td><td>14 chars</td></tr>
        <tr><td>16.001</td><td>Latin-1 string</td><td>14 chars</td></tr>
      </table>
    `,
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // BEST PRACTICES
  // ═══════════════════════════════════════════════════════════════════════════
  bestPractices: {
    title: "Best Practices",
    content: `
      <h4>Group Address Planning</h4>
      <ul>
        <li><strong>Plan before programming</strong> — Sketch your GA structure on paper first</li>
        <li><strong>Leave gaps</strong> — Don't use consecutive addresses; leave room for expansion</li>
        <li><strong>Separate commands and status</strong> — Use different GAs for control vs feedback</li>
        <li><strong>Be consistent</strong> — Use the same pattern throughout the project</li>
        <li><strong>Document everything</strong> — ETS project notes, spreadsheets, diagrams</li>
      </ul>

      <h4>Status Feedback</h4>
      <ul>
        <li>Always use status GAs for actuators — essential for visualization</li>
        <li>Status should be on a different GA than command</li>
        <li>Configure actuators to send status on change</li>
        <li>Consider using "Read on Init" for startup synchronization</li>
      </ul>

      <h4>Central Functions</h4>
      <ul>
        <li>Use dedicated "All Off" or "All Blinds Up" addresses</li>
        <li>Link multiple actuators to the same GA for group control</li>
        <li>Consider scenes for complex multi-device actions</li>
      </ul>

      <h4>Line Planning</h4>
      <ul>
        <li>Max 64 devices per line (recommended), 256 absolute max</li>
        <li>Group devices by location or function on each line</li>
        <li>Keep heavy traffic devices (sensors) on separate lines</li>
        <li>Plan for future expansion</li>
      </ul>

      <h4>Testing & Commissioning</h4>
      <ul>
        <li>Test each device individually before linking</li>
        <li>Use ETS bus monitor to verify telegrams</li>
        <li>Check for address conflicts</li>
        <li>Verify all status feedback works</li>
        <li>Test edge cases (rapid switching, simultaneous commands)</li>
      </ul>
    `,
  },
};

/**
 * Generate HTML for the reference panel
 */
export function generateReferenceHTML() {
  const sections = Object.values(KNX_REFERENCE);
  
  return `
    <div class="ref-nav">
      ${sections.map((s, i) => `<button class="ref-nav-btn" data-section="${i}">${s.title}</button>`).join('')}
    </div>
    <div class="ref-content">
      ${sections.map((s, i) => `
        <div class="ref-section" data-section="${i}" ${i === 0 ? '' : 'style="display:none"'}>
          <h3>${s.title}</h3>
          ${s.content}
        </div>
      `).join('')}
    </div>
  `;
}
