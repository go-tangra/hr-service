import './styles/tailwind.css';
import type { TangraModule } from './sdk';
import routes from './routes';
import { useHrAbsenceTypeStore } from './stores/hr-absence-type.state';
import { useHrLeaveStore } from './stores/hr-leave.state';
import { useHrAllowanceStore } from './stores/hr-allowance.state';
import { useHrSystemStore } from './stores/hr-system.state';
import enUS from './locales/en-US.json';

const hrModule: TangraModule = {
  id: 'hr',
  version: '1.0.0',
  routes,
  stores: {
    'hr-absence-type': useHrAbsenceTypeStore,
    'hr-leave': useHrLeaveStore,
    'hr-allowance': useHrAllowanceStore,
    'hr-system': useHrSystemStore,
  },
  locales: {
    'en-US': enUS,
  },
};

export default hrModule;
