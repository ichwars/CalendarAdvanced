import { DatePicker } from './DatePicker';
import { normalizeTime, TimePicker } from './TimePicker';

interface DateTimePickerProps {
  disabled?: boolean;
  label: string;
  minuteStep?: number;
  name: string;
  onChange: (value: string) => void;
  timeDisabled?: boolean;
  value: string;
}

export function DateTimePicker({ disabled = false, label, minuteStep, name, onChange, timeDisabled = false, value }: DateTimePickerProps) {
  const [dateValue, timeValue] = splitDateTime(value);

  function updateDate(nextDate: string) {
    onChange(`${nextDate}T${timeValue}`);
  }

  function updateTime(nextTime: string) {
    onChange(`${dateValue}T${normalizeTime(nextTime)}`);
  }

  return (
    <div className={disabled ? 'field field--disabled' : 'field'}>
      <span>{label}</span>
      <div className="datetime-picker">
        <DatePicker ariaLabel={label} disabled={disabled} value={dateValue} onChange={updateDate} />
        <TimePicker ariaLabel={label} disabled={disabled || timeDisabled} minuteStep={minuteStep} name={`${name}-time`} value={timeValue} onChange={updateTime} />
      </div>
    </div>
  );
}

function splitDateTime(value: string): [string, string] {
  const [dateValue, timeValue = '00:00'] = value.split('T');
  return [dateValue, normalizeTime(timeValue)];
}
