/// Static metadata for common KNX device types.
/// Maps device type → domain, capabilities, and expected address functions.
/// Used by the "Add Device" form to provide smart defaults.

/// An address function expected for a device type.
class AddressFunction {
  final String key;
  final String label;
  final bool required;
  final String dpt;
  final List<String> flags;

  const AddressFunction(this.key, this.label, {
    this.required = false,
    this.dpt = '',
    this.flags = const [],
  });
}

/// Metadata for a device type.
class DeviceTypeInfo {
  final String type;
  final String label;
  final String category;
  final String domain;
  final List<String> defaultCapabilities;
  final List<AddressFunction> addressFunctions;

  const DeviceTypeInfo({
    required this.type,
    required this.label,
    required this.category,
    required this.domain,
    this.defaultCapabilities = const [],
    this.addressFunctions = const [],
  });
}

/// All supported device types grouped by category.
const kDeviceTypes = <DeviceTypeInfo>[
  // -- Lighting --
  DeviceTypeInfo(
    type: 'light_switch',
    label: 'Light Switch',
    category: 'Lighting',
    domain: 'lighting',
    defaultCapabilities: ['on_off'],
    addressFunctions: [
      AddressFunction('switch', 'Switch', required: true, dpt: '1.001', flags: ['write']),
      AddressFunction('switch_status', 'Switch Status', dpt: '1.001', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'light_dimmer',
    label: 'Light Dimmer',
    category: 'Lighting',
    domain: 'lighting',
    defaultCapabilities: ['on_off', 'dim'],
    addressFunctions: [
      AddressFunction('switch', 'Switch', required: true, dpt: '1.001', flags: ['write']),
      AddressFunction('switch_status', 'Switch Status', dpt: '1.001', flags: ['read', 'transmit']),
      AddressFunction('brightness', 'Brightness', dpt: '5.001', flags: ['write']),
      AddressFunction('brightness_status', 'Brightness Status', dpt: '5.001', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'light_ct',
    label: 'Light Colour Temp',
    category: 'Lighting',
    domain: 'lighting',
    defaultCapabilities: ['on_off', 'dim', 'color_temp'],
    addressFunctions: [
      AddressFunction('switch', 'Switch', required: true, dpt: '1.001', flags: ['write']),
      AddressFunction('switch_status', 'Switch Status', dpt: '1.001', flags: ['read', 'transmit']),
      AddressFunction('brightness', 'Brightness', dpt: '5.001', flags: ['write']),
      AddressFunction('brightness_status', 'Brightness Status', dpt: '5.001', flags: ['read', 'transmit']),
      AddressFunction('color_temperature', 'Colour Temperature', dpt: '7.600', flags: ['write']),
      AddressFunction('color_temperature_status', 'Colour Temp Status', dpt: '7.600', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'light_rgb',
    label: 'Light RGB',
    category: 'Lighting',
    domain: 'lighting',
    defaultCapabilities: ['on_off', 'dim', 'color_rgb'],
    addressFunctions: [
      AddressFunction('switch', 'Switch', required: true, dpt: '1.001', flags: ['write']),
      AddressFunction('switch_status', 'Switch Status', dpt: '1.001', flags: ['read', 'transmit']),
      AddressFunction('brightness', 'Brightness', dpt: '5.001', flags: ['write']),
      AddressFunction('rgb', 'RGB Colour', dpt: '232.600', flags: ['write']),
      AddressFunction('rgb_status', 'RGB Status', dpt: '232.600', flags: ['read', 'transmit']),
    ],
  ),

  // -- Climate --
  DeviceTypeInfo(
    type: 'thermostat',
    label: 'Thermostat',
    category: 'Climate',
    domain: 'climate',
    defaultCapabilities: ['temperature_read', 'temperature_set'],
    addressFunctions: [
      AddressFunction('temperature', 'Actual Temperature', required: true, dpt: '9.001', flags: ['read', 'transmit']),
      AddressFunction('setpoint', 'Setpoint', dpt: '9.001', flags: ['write', 'read']),
      AddressFunction('heating_output', 'Heating Output', dpt: '5.001', flags: ['write']),
      AddressFunction('hvac_mode', 'HVAC Mode', dpt: '20.102', flags: ['write']),
    ],
  ),
  DeviceTypeInfo(
    type: 'heating_actuator',
    label: 'Heating Actuator',
    category: 'Climate',
    domain: 'climate',
    defaultCapabilities: ['on_off'],
    addressFunctions: [
      AddressFunction('valve', 'Valve Command', required: true, dpt: '5.001', flags: ['write']),
      AddressFunction('valve_status', 'Valve Status', dpt: '5.001', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'valve_actuator',
    label: 'Valve Actuator',
    category: 'Climate',
    domain: 'climate',
    defaultCapabilities: ['on_off'],
    addressFunctions: [
      AddressFunction('valve', 'Valve Command', required: true, dpt: '5.001', flags: ['write']),
      AddressFunction('valve_status', 'Valve Status', dpt: '5.001', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'temperature_sensor',
    label: 'Temperature Sensor',
    category: 'Climate',
    domain: 'sensor',
    defaultCapabilities: ['temperature_read'],
    addressFunctions: [
      AddressFunction('temperature', 'Temperature', required: true, dpt: '9.001', flags: ['read', 'transmit']),
    ],
  ),

  // -- Blinds --
  DeviceTypeInfo(
    type: 'blind_switch',
    label: 'Blind Switch',
    category: 'Blinds',
    domain: 'blinds',
    defaultCapabilities: ['on_off'],
    addressFunctions: [
      AddressFunction('move', 'Move Up/Down', required: true, dpt: '1.008', flags: ['write']),
      AddressFunction('stop', 'Stop', dpt: '1.007', flags: ['write']),
    ],
  ),
  DeviceTypeInfo(
    type: 'blind_position',
    label: 'Blind Position',
    category: 'Blinds',
    domain: 'blinds',
    defaultCapabilities: ['position'],
    addressFunctions: [
      AddressFunction('position', 'Position', required: true, dpt: '5.001', flags: ['write']),
      AddressFunction('position_status', 'Position Status', dpt: '5.001', flags: ['read', 'transmit']),
      AddressFunction('move', 'Move Up/Down', dpt: '1.008', flags: ['write']),
      AddressFunction('stop', 'Stop', dpt: '1.007', flags: ['write']),
    ],
  ),
  DeviceTypeInfo(
    type: 'blind_tilt',
    label: 'Blind with Tilt',
    category: 'Blinds',
    domain: 'blinds',
    defaultCapabilities: ['position', 'tilt'],
    addressFunctions: [
      AddressFunction('position', 'Position', required: true, dpt: '5.001', flags: ['write']),
      AddressFunction('position_status', 'Position Status', dpt: '5.001', flags: ['read', 'transmit']),
      AddressFunction('slat', 'Slat/Tilt Angle', dpt: '5.001', flags: ['write']),
      AddressFunction('slat_status', 'Slat Status', dpt: '5.001', flags: ['read', 'transmit']),
      AddressFunction('move', 'Move Up/Down', dpt: '1.008', flags: ['write']),
      AddressFunction('stop', 'Stop', dpt: '1.007', flags: ['write']),
    ],
  ),

  // -- Sensors --
  DeviceTypeInfo(
    type: 'presence_sensor',
    label: 'Presence Sensor',
    category: 'Sensors',
    domain: 'sensor',
    defaultCapabilities: ['presence_detect'],
    addressFunctions: [
      AddressFunction('presence', 'Presence', required: true, dpt: '1.018', flags: ['read', 'transmit']),
      AddressFunction('lux', 'Lux Level', dpt: '9.004', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'motion_sensor',
    label: 'Motion Sensor',
    category: 'Sensors',
    domain: 'sensor',
    defaultCapabilities: ['motion_detect'],
    addressFunctions: [
      AddressFunction('presence', 'Motion', required: true, dpt: '1.018', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'light_sensor',
    label: 'Light Sensor',
    category: 'Sensors',
    domain: 'sensor',
    defaultCapabilities: ['light_level_read'],
    addressFunctions: [
      AddressFunction('lux', 'Lux Level', required: true, dpt: '9.004', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'humidity_sensor',
    label: 'Humidity Sensor',
    category: 'Sensors',
    domain: 'sensor',
    defaultCapabilities: ['humidity_read'],
    addressFunctions: [
      AddressFunction('humidity', 'Humidity', required: true, dpt: '9.007', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'co2_sensor',
    label: 'CO2 Sensor',
    category: 'Sensors',
    domain: 'sensor',
    defaultCapabilities: ['co2_read'],
    addressFunctions: [
      AddressFunction('co2', 'CO2 Level', required: true, dpt: '9.008', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'door_sensor',
    label: 'Door Sensor',
    category: 'Sensors',
    domain: 'sensor',
    defaultCapabilities: ['contact_state'],
    addressFunctions: [
      AddressFunction('open_close', 'Contact State', required: true, dpt: '1.009', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'window_sensor',
    label: 'Window Sensor',
    category: 'Sensors',
    domain: 'sensor',
    defaultCapabilities: ['contact_state'],
    addressFunctions: [
      AddressFunction('open_close', 'Contact State', required: true, dpt: '1.009', flags: ['read', 'transmit']),
    ],
  ),

  // -- Energy --
  DeviceTypeInfo(
    type: 'energy_meter',
    label: 'Energy Meter',
    category: 'Energy',
    domain: 'energy',
    defaultCapabilities: ['energy_read', 'power_read'],
    addressFunctions: [
      AddressFunction('active_energy', 'Total Energy (kWh)', required: true, dpt: '13.010', flags: ['read', 'transmit']),
      AddressFunction('power', 'Active Power (W)', dpt: '14.056', flags: ['read', 'transmit']),
      AddressFunction('voltage', 'Voltage', dpt: '14.027', flags: ['read', 'transmit']),
      AddressFunction('current', 'Current', dpt: '14.019', flags: ['read', 'transmit']),
    ],
  ),

  // -- Infrastructure --
  DeviceTypeInfo(
    type: 'switch_actuator',
    label: 'Switch Actuator',
    category: 'Infrastructure',
    domain: 'infrastructure',
    defaultCapabilities: ['on_off'],
    addressFunctions: [
      AddressFunction('switch', 'Switch', required: true, dpt: '1.001', flags: ['write']),
      AddressFunction('switch_status', 'Switch Status', dpt: '1.001', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'push_button',
    label: 'Push Button',
    category: 'Controls',
    domain: 'lighting',
    defaultCapabilities: ['on_off'],
    addressFunctions: [
      AddressFunction('button_1', 'Button Press', required: true, dpt: '1.001', flags: ['write', 'transmit']),
      AddressFunction('button_1_led', 'LED Status', dpt: '1.001', flags: ['write']),
    ],
  ),
  DeviceTypeInfo(
    type: 'binary_input',
    label: 'Binary Input',
    category: 'Controls',
    domain: 'sensor',
    defaultCapabilities: ['contact_state'],
    addressFunctions: [
      AddressFunction('open_close', 'Input State', required: true, dpt: '1.009', flags: ['read', 'transmit']),
    ],
  ),
  DeviceTypeInfo(
    type: 'scene_controller',
    label: 'Scene Controller',
    category: 'Controls',
    domain: 'lighting',
    defaultCapabilities: ['on_off'],
    addressFunctions: [
      AddressFunction('scene_number', 'Scene Number', required: true, dpt: '17.001', flags: ['write', 'transmit']),
    ],
  ),
];

/// Lookup a device type info by type string.
DeviceTypeInfo? getDeviceTypeInfo(String type) {
  for (final info in kDeviceTypes) {
    if (info.type == type) return info;
  }
  return null;
}

/// Get all unique categories in display order.
List<String> getDeviceCategories() {
  final seen = <String>{};
  final result = <String>[];
  for (final info in kDeviceTypes) {
    if (seen.add(info.category)) result.add(info.category);
  }
  return result;
}

/// Get device types for a given category.
List<DeviceTypeInfo> getDeviceTypesForCategory(String category) {
  return kDeviceTypes.where((t) => t.category == category).toList();
}
