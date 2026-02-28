<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue';
import { Button, Select, SelectOption, Tag, Tooltip } from 'ant-design-vue';
import { Page, useVbenModal } from 'shell/vben/common-ui';
import { $t } from 'shell/locales';
import { useHrLeaveStore } from '../../stores/hr-leave.state';
import { useHrAbsenceTypeStore } from '../../stores/hr-absence-type.state';
import type { CalendarEvent, AbsenceType } from '../../api/services';
import { requestClient } from 'shell/api/request';
import RequestDrawer from '../request/request-drawer.vue';

type ViewMode = 'week' | '2weeks' | 'month';

interface PortalUser {
  id: number;
  username?: string;
  realname?: string;
  orgUnitNames?: string[];
}

const leaveStore = useHrLeaveStore();
const absenceTypeStore = useHrAbsenceTypeStore();

// --- State ---
const viewMode = ref<ViewMode>('month');
const currentDate = ref(new Date());
const users = ref<PortalUser[]>([]);
const absenceTypes = ref<AbsenceType[]>([]);
const events = ref<CalendarEvent[]>([]);
const selectedOrgUnit = ref<string>();
const orgUnits = ref<string[]>([]);
const collapsedOrgUnits = ref<Set<string>>(new Set());

// Drag state
const isDragging = ref(false);
const dragUserId = ref(0);
const dragStartCol = ref(-1);
const dragCurrentCol = ref(-1);

// Resize state
const isResizing = ref(false);
const resizeEvent = ref<CalendarEvent | null>(null);
const resizeEdge = ref<'left' | 'right'>('right');
const resizeOrigStart = ref(0);
const resizeOrigEnd = ref(0);
const resizeCurrent = ref(0);

// Tooltip state
const hoveredBar = ref<CalendarEvent | null>(null);
const tooltipPos = ref({ x: 0, y: 0 });

// --- Constants ---
const DAY_WIDTHS: Record<ViewMode, number> = { week: 140, '2weeks': 80, month: 44 };
const ROW_HEIGHT = 52;
const DEPT_ROW_HEIGHT = 36;
const EMP_COL_WIDTH = 232;
const HEADER_ROW_HEIGHT = 52;

const dayWidth = computed(() => DAY_WIDTHS[viewMode.value]);

// --- Date helpers ---
function toISODate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function startOfWeek(d: Date): Date {
  const copy = new Date(d.getFullYear(), d.getMonth(), d.getDate());
  const dow = copy.getDay();
  copy.setDate(copy.getDate() - dow + (dow === 0 ? -6 : 1));
  return copy;
}

const viewStart = computed(() => {
  const d = new Date(currentDate.value);
  if (viewMode.value === 'month') return new Date(d.getFullYear(), d.getMonth(), 1);
  return startOfWeek(d);
});

const numDays = computed(() => {
  if (viewMode.value === 'week') return 7;
  if (viewMode.value === '2weeks') return 14;
  return new Date(viewStart.value.getFullYear(), viewStart.value.getMonth() + 1, 0).getDate();
});

const todayDate = computed(() => {
  const d = new Date();
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
});

interface DayInfo {
  date: Date;
  dayOfWeek: string;
  dayNum: number;
  isWeekend: boolean;
  isToday: boolean;
  iso: string;
}

const days = computed<DayInfo[]>(() => {
  const result: DayInfo[] = [];
  for (let i = 0; i < numDays.value; i++) {
    const d = new Date(viewStart.value);
    d.setDate(d.getDate() + i);
    const dow = d.getDay();
    result.push({
      date: new Date(d),
      dayOfWeek: d.toLocaleDateString('en-US', { weekday: 'short' }),
      dayNum: d.getDate(),
      isWeekend: dow === 0 || dow === 6,
      isToday: d.getTime() === todayDate.value.getTime(),
      iso: toISODate(d),
    });
  }
  return result;
});

// Compute month spans for the sub-header
const monthSpans = computed(() => {
  const spans: { label: string; cols: number }[] = [];
  let currentLabel = '';
  let count = 0;
  for (const day of days.value) {
    const label = day.date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
    if (label !== currentLabel) {
      if (currentLabel) spans.push({ label: currentLabel, cols: count });
      currentLabel = label;
      count = 1;
    } else {
      count++;
    }
  }
  if (currentLabel) spans.push({ label: currentLabel, cols: count });
  return spans;
});

const headerTitle = computed(() => {
  const d = viewStart.value;
  if (viewMode.value === 'month') {
    return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
  }
  const end = new Date(d);
  end.setDate(end.getDate() + numDays.value - 1);
  if (d.getMonth() === end.getMonth()) {
    return d.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
  }
  return `${d.toLocaleDateString('en-US', { month: 'short' })} – ${end.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })}`;
});

// --- Navigation ---
function navigate(dir: number) {
  const d = new Date(currentDate.value);
  if (viewMode.value === 'month') d.setMonth(d.getMonth() + dir);
  else if (viewMode.value === '2weeks') d.setDate(d.getDate() + dir * 14);
  else d.setDate(d.getDate() + dir * 7);
  currentDate.value = d;
}

function goToday() {
  currentDate.value = new Date();
}

// --- Data loading ---
async function loadUsers() {
  try {
    const resp = await requestClient.get<{ items: PortalUser[] }>('/admin/v1/users', { params: { noPaging: true } });
    users.value = resp.items || [];
    const unitSet = new Set<string>();
    for (const user of users.value) {
      if (user.orgUnitNames) {
        for (const name of user.orgUnitNames) {
          unitSet.add(name);
        }
      }
    }
    orgUnits.value = Array.from(unitSet).sort();
  } catch { /* noop */ }
}

async function loadAbsenceTypes() {
  try {
    const resp = await absenceTypeStore.listAbsenceTypes(undefined, null);
    absenceTypes.value = resp.items || [];
  } catch { /* noop */ }
}

async function loadEvents() {
  try {
    const startStr = toISODate(viewStart.value);
    const end = new Date(viewStart.value);
    end.setDate(end.getDate() + numDays.value);
    const resp = await leaveStore.getCalendarEvents({
      startDate: startStr,
      endDate: toISODate(end),
      orgUnitName: selectedOrgUnit.value,
    });
    events.value = resp.events || [];
  } catch { /* noop */ }
}

// --- User grouping ---
function getUserDisplayName(user: PortalUser): string {
  return user.realname || user.username || '';
}

function getUserInitials(user: PortalUser): string {
  const name = getUserDisplayName(user);
  const parts = name.split(/\s+/);
  if (parts.length >= 2) {
    return (parts[0]?.[0] || '') + (parts[1]?.[0] || '');
  }
  return name.substring(0, 2);
}

const groupedRows = computed(() => {
  let filtered = users.value;
  if (selectedOrgUnit.value) {
    filtered = filtered.filter((u) => u.orgUnitNames?.includes(selectedOrgUnit.value!));
  }

  const byOrgUnit = new Map<string, PortalUser[]>();
  for (const user of filtered) {
    const unit = user.orgUnitNames?.[0] || 'Unassigned';
    if (!byOrgUnit.has(unit)) byOrgUnit.set(unit, []);
    byOrgUnit.get(unit)!.push(user);
  }

  type Row =
    | { type: 'dept'; department: string; count: number }
    | { type: 'user'; user: PortalUser };

  const result: Row[] = [];
  const sortedUnits = Array.from(byOrgUnit.keys()).sort();
  for (const unit of sortedUnits) {
    const unitUsers = byOrgUnit.get(unit)!;
    result.push({ type: 'dept', department: unit, count: unitUsers.length });
    if (!collapsedOrgUnits.value.has(unit)) {
      for (const user of unitUsers) {
        result.push({ type: 'user', user });
      }
    }
  }
  return result;
});

// --- Bar computation ---
function dateToDayIndex(dateStr: string): number {
  const parts = dateStr.split('T')[0]!.split('-');
  const d = new Date(+parts[0]!, +parts[1]! - 1, +parts[2]!);
  return Math.round((d.getTime() - viewStart.value.getTime()) / 86400000);
}

interface BarInfo {
  event: CalendarEvent;
  left: number;
  width: number;
  isPending: boolean;
  startIdx: number;
  endIdx: number;
}

const barsByUser = computed(() => {
  const map = new Map<number, BarInfo[]>();
  for (const evt of events.value) {
    if (
      evt.status === 'LEAVE_REQUEST_STATUS_CANCELLED' ||
      evt.status === 'LEAVE_REQUEST_STATUS_REJECTED'
    )
      continue;
    if (!evt.startDate || !evt.endDate || !evt.userId) continue;

    let startIdx = dateToDayIndex(evt.startDate);
    let endIdx = dateToDayIndex(evt.endDate);
    if (endIdx < 0 || startIdx >= numDays.value) continue;
    startIdx = Math.max(0, startIdx);
    endIdx = Math.min(numDays.value - 1, endIdx);

    const dw = dayWidth.value;
    const left = startIdx * dw;
    const width = (endIdx - startIdx + 1) * dw;
    const isPending = evt.status === 'LEAVE_REQUEST_STATUS_PENDING';

    if (!map.has(evt.userId)) map.set(evt.userId, []);
    map.get(evt.userId)!.push({ event: evt, left, width, isPending, startIdx, endIdx });
  }
  return map;
});

// --- Today marker ---
const todayMarkerLeft = computed(() => {
  const idx = days.value.findIndex((d) => d.isToday);
  if (idx < 0) return null;
  return idx * dayWidth.value + dayWidth.value / 2;
});

// --- Drag-to-create ---
function onCellMouseDown(userId: number, dayIndex: number, e: MouseEvent) {
  if (e.button !== 0 || isResizing.value) return;
  isDragging.value = true;
  dragUserId.value = userId;
  dragStartCol.value = dayIndex;
  dragCurrentCol.value = dayIndex;
  e.preventDefault();
}

function onCellMouseEnter(dayIndex: number) {
  if (isDragging.value) dragCurrentCol.value = dayIndex;
  if (isResizing.value) resizeCurrent.value = dayIndex;
}

function isDragHighlighted(userId: number, dayIndex: number): boolean {
  if (!isDragging.value || dragUserId.value !== userId) return false;
  const s = Math.min(dragStartCol.value, dragCurrentCol.value);
  const e = Math.max(dragStartCol.value, dragCurrentCol.value);
  return dayIndex >= s && dayIndex <= e;
}

function dragSelectionStyle(userId: number): Record<string, string> | null {
  if (!isDragging.value || dragUserId.value !== userId) return null;
  const s = Math.min(dragStartCol.value, dragCurrentCol.value);
  const e = Math.max(dragStartCol.value, dragCurrentCol.value);
  return {
    left: `${s * dayWidth.value}px`,
    width: `${(e - s + 1) * dayWidth.value}px`,
  };
}

// --- Bar resize ---
function onResizeStart(evt: CalendarEvent, edge: 'left' | 'right', e: MouseEvent) {
  e.stopPropagation();
  e.preventDefault();
  isResizing.value = true;
  resizeEvent.value = evt;
  resizeEdge.value = edge;
  const startIdx = dateToDayIndex(evt.startDate || '');
  const endIdx = dateToDayIndex(evt.endDate || '');
  resizeOrigStart.value = Math.max(0, startIdx);
  resizeOrigEnd.value = Math.min(numDays.value - 1, endIdx);
  resizeCurrent.value = edge === 'left' ? resizeOrigStart.value : resizeOrigEnd.value;
}

function resizedBarStyle(userId: number): Record<string, string> | null {
  if (!isResizing.value || resizeEvent.value?.userId !== userId) return null;
  const s =
    resizeEdge.value === 'left'
      ? Math.min(resizeCurrent.value, resizeOrigEnd.value)
      : resizeOrigStart.value;
  const e =
    resizeEdge.value === 'right'
      ? Math.max(resizeCurrent.value, resizeOrigStart.value)
      : resizeOrigEnd.value;
  return {
    left: `${s * dayWidth.value}px`,
    width: `${(e - s + 1) * dayWidth.value}px`,
    backgroundColor: resizeEvent.value?.color || '#4096ff',
    opacity: '0.5',
  };
}

// --- Mouse up handler ---
function onGlobalMouseUp() {
  if (isDragging.value) {
    isDragging.value = false;
    const s = Math.min(dragStartCol.value, dragCurrentCol.value);
    const e = Math.max(dragStartCol.value, dragCurrentCol.value);
    if (s >= 0 && e >= 0 && days.value[s] && days.value[e]) {
      openCreateDrawer(dragUserId.value, days.value[s]!.iso, days.value[e]!.iso);
    }
  }
  if (isResizing.value && resizeEvent.value) {
    const s =
      resizeEdge.value === 'left'
        ? Math.min(resizeCurrent.value, resizeOrigEnd.value)
        : resizeOrigStart.value;
    const e =
      resizeEdge.value === 'right'
        ? Math.max(resizeCurrent.value, resizeOrigStart.value)
        : resizeOrigEnd.value;
    if (days.value[s] && days.value[e]) {
      updateRequestDates(resizeEvent.value.id, days.value[s]!.iso, days.value[e]!.iso);
    }
    isResizing.value = false;
    resizeEvent.value = null;
  }
}

async function updateRequestDates(id: string, startDate: string, endDate: string) {
  try {
    await leaveStore.updateLeaveRequest(id, { startDate, endDate }, ['start_date', 'end_date']);
    loadEvents();
  } catch { /* noop */ }
}

// --- Request drawer ---
const [RequestDrawerComponent, requestDrawerApi] = useVbenModal({
  connectedComponent: RequestDrawer,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) loadEvents();
  },
});

function openCreateDrawer(userId?: number, startDate?: string, endDate?: string) {
  const user = userId ? users.value.find((u) => u.id === userId) : undefined;
  requestDrawerApi.setData({
    row: {
      userId,
      userName: user ? getUserDisplayName(user) : undefined,
      orgUnitName: user?.orgUnitNames?.[0],
      startDate,
      endDate,
    } as any,
    mode: 'create',
  });
  requestDrawerApi.open();
}

function openViewDrawer(evt: CalendarEvent) {
  requestDrawerApi.setData({ row: evt, mode: 'view' });
  requestDrawerApi.open();
}

function handleNewRequest() {
  requestDrawerApi.setData({ row: {}, mode: 'create' });
  requestDrawerApi.open();
}

// --- Bar tooltip ---
function onBarMouseEnter(evt: CalendarEvent, e: MouseEvent) {
  hoveredBar.value = evt;
  const rect = (e.target as HTMLElement).getBoundingClientRect();
  tooltipPos.value = { x: rect.left + rect.width / 2, y: rect.top - 8 };
}

function onBarMouseLeave() {
  hoveredBar.value = null;
}

// --- Department toggle ---
function toggleDept(dept: string) {
  const s = new Set(collapsedOrgUnits.value);
  if (s.has(dept)) s.delete(dept);
  else s.add(dept);
  collapsedOrgUnits.value = s;
}

// --- View switcher ---
const viewOptions = computed(() => [
  { value: 'week', label: $t('hr.page.calendar.viewWeek') },
  { value: '2weeks', label: $t('hr.page.calendar.view2Weeks') },
  { value: 'month', label: $t('hr.page.calendar.viewMonth') },
]);

// --- Lifecycle ---
watch([viewMode, currentDate, selectedOrgUnit], () => loadEvents());

onMounted(() => {
  loadUsers();
  loadAbsenceTypes();
  loadEvents();
  document.addEventListener('mouseup', onGlobalMouseUp);
});

onUnmounted(() => {
  document.removeEventListener('mouseup', onGlobalMouseUp);
});
</script>

<template>
  <Page auto-content-height>
    <!-- Toolbar -->
    <div class="tl-toolbar">
      <div class="tl-toolbar-left">
        <div class="tl-nav-group">
          <button class="tl-nav-btn" @click="navigate(-1)">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M10 12L6 8L10 4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button class="tl-today-btn" @click="goToday">
            {{ $t('hr.page.calendar.today') }}
          </button>
          <button class="tl-nav-btn" @click="navigate(1)">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <path d="M6 4L10 8L6 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
        <h2 class="tl-title">{{ headerTitle }}</h2>
      </div>

      <div class="tl-toolbar-center">
        <div class="tl-view-switcher">
          <button
            v-for="opt in viewOptions"
            :key="opt.value"
            class="tl-view-btn"
            :class="{ active: viewMode === opt.value }"
            @click="viewMode = opt.value as ViewMode"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <div class="tl-toolbar-right">
        <Select
          v-model:value="selectedOrgUnit"
          :placeholder="$t('hr.page.calendar.filterOrgUnit')"
          allow-clear
          style="width: 180px"
          size="middle"
        >
          <SelectOption v-for="unit in orgUnits" :key="unit" :value="unit">
            {{ unit }}
          </SelectOption>
        </Select>
        <Button type="primary" @click="handleNewRequest">
          + {{ $t('hr.page.calendar.newRequest') }}
        </Button>
      </div>
    </div>

    <!-- Timeline -->
    <div class="tl-container" :class="{ 'is-dragging': isDragging || isResizing }">
      <div
        class="tl-scroll"
        :style="{ minWidth: `${EMP_COL_WIDTH + numDays * dayWidth}px` }"
      >
        <!-- Header: month spans -->
        <div class="tl-header-row tl-header-months">
          <div class="tl-header-corner" :style="{ width: `${EMP_COL_WIDTH}px`, minWidth: `${EMP_COL_WIDTH}px` }">
            <span class="tl-emp-header">{{ $t('hr.page.calendar.users') }}</span>
          </div>
          <div
            v-for="(span, si) in monthSpans"
            :key="si"
            class="tl-month-span"
            :style="{ width: `${span.cols * dayWidth}px` }"
          >
            {{ span.label }}
          </div>
        </div>

        <!-- Header: day cells -->
        <div class="tl-header-row tl-header-days">
          <div class="tl-header-corner" :style="{ width: `${EMP_COL_WIDTH}px`, minWidth: `${EMP_COL_WIDTH}px` }" />
          <div
            v-for="(day, di) in days"
            :key="di"
            class="tl-day-header"
            :class="{ weekend: day.isWeekend, today: day.isToday }"
            :style="{ width: `${dayWidth}px`, minWidth: `${dayWidth}px` }"
          >
            <span class="tl-day-name">{{ day.dayOfWeek }}</span>
            <span class="tl-day-num" :class="{ 'today-badge': day.isToday }">{{ day.dayNum }}</span>
          </div>
        </div>

        <!-- Body rows -->
        <template v-for="(row, ri) in groupedRows" :key="ri">
          <!-- Department header -->
          <div
            v-if="row.type === 'dept'"
            class="tl-dept-row"
            @click="toggleDept(row.department)"
          >
            <div class="tl-dept-label" :style="{ width: `${EMP_COL_WIDTH}px`, minWidth: `${EMP_COL_WIDTH}px` }">
              <svg
                class="tl-dept-chevron"
                :class="{ collapsed: collapsedOrgUnits.has(row.department) }"
                width="14" height="14" viewBox="0 0 14 14" fill="none"
              >
                <path d="M5 3L9 7L5 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <span>{{ row.department }}</span>
              <span class="tl-dept-count">{{ row.count }}</span>
            </div>
            <div class="tl-dept-fill" />
          </div>

          <!-- User row -->
          <div v-else class="tl-emp-row">
            <div class="tl-emp-cell" :style="{ width: `${EMP_COL_WIDTH}px`, minWidth: `${EMP_COL_WIDTH}px` }">
              <div class="tl-avatar" :style="{ backgroundColor: stringToColor(getUserDisplayName(row.user)) }">
                {{ getUserInitials(row.user) }}
              </div>
              <div class="tl-emp-info">
                <div class="tl-emp-name">
                  {{ getUserDisplayName(row.user) }}
                </div>
              </div>
            </div>

            <div class="tl-row-timeline" :style="{ width: `${numDays * dayWidth}px` }">
              <!-- Day cell backgrounds -->
              <div
                v-for="(day, di) in days"
                :key="di"
                class="tl-cell"
                :class="{
                  weekend: day.isWeekend,
                  today: day.isToday,
                  'drag-highlight': isDragHighlighted(row.user.id, di),
                }"
                :style="{ width: `${dayWidth}px`, minWidth: `${dayWidth}px` }"
                @mousedown="onCellMouseDown(row.user.id, di, $event)"
                @mouseenter="onCellMouseEnter(di)"
              />

              <!-- Drag selection ghost -->
              <div
                v-if="dragSelectionStyle(row.user.id)"
                class="tl-drag-ghost"
                :style="dragSelectionStyle(row.user.id)!"
              />

              <!-- Resize ghost -->
              <div
                v-if="resizedBarStyle(row.user.id)"
                class="tl-resize-ghost"
                :style="resizedBarStyle(row.user.id)!"
              />

              <!-- Leave request bars -->
              <div
                v-for="bar in barsByUser.get(row.user.id) || []"
                :key="bar.event.id"
                class="tl-bar"
                :class="{ pending: bar.isPending }"
                :style="{
                  left: `${bar.left + 2}px`,
                  width: `${bar.width - 4}px`,
                  '--bar-color': bar.event.color || '#4096ff',
                }"
                @click.stop="openViewDrawer(bar.event)"
                @mouseenter="onBarMouseEnter(bar.event, $event)"
                @mouseleave="onBarMouseLeave"
              >
                <!-- Resize handles -->
                <div
                  v-if="bar.isPending"
                  class="tl-bar-handle tl-bar-handle-left"
                  @mousedown="onResizeStart(bar.event, 'left', $event)"
                />
                <span class="tl-bar-label">
                  {{ bar.event.absenceTypeName }}
                  <template v-if="bar.width > 120"> · {{ bar.event.days }}d</template>
                </span>
                <div
                  v-if="bar.isPending"
                  class="tl-bar-handle tl-bar-handle-right"
                  @mousedown="onResizeStart(bar.event, 'right', $event)"
                />
              </div>

              <!-- Today marker line -->
              <div
                v-if="todayMarkerLeft !== null"
                class="tl-today-line"
                :style="{ left: `${todayMarkerLeft}px` }"
              />
            </div>
          </div>
        </template>

        <!-- Empty state -->
        <div v-if="groupedRows.length === 0" class="tl-empty">
          {{ $t('hr.page.calendar.noUsers') }}
        </div>
      </div>
    </div>

    <!-- Floating tooltip -->
    <Teleport to="body">
      <div
        v-if="hoveredBar"
        class="tl-tooltip"
        :style="{
          left: `${tooltipPos.x}px`,
          top: `${tooltipPos.y}px`,
        }"
      >
        <div class="tl-tooltip-name">{{ hoveredBar.userName }}</div>
        <div class="tl-tooltip-type">
          <span class="tl-tooltip-dot" :style="{ backgroundColor: hoveredBar.color || '#4096ff' }" />
          {{ hoveredBar.absenceTypeName }}
        </div>
        <div class="tl-tooltip-dates">{{ hoveredBar.startDate }} → {{ hoveredBar.endDate }}</div>
        <div class="tl-tooltip-meta">
          <span>{{ hoveredBar.days }} {{ $t('hr.page.request.days') }}</span>
          <Tag
            :color="hoveredBar.status === 'LEAVE_REQUEST_STATUS_APPROVED' ? 'green' : 'orange'"
            size="small"
            class="ml-1"
          >
            {{ hoveredBar.status === 'LEAVE_REQUEST_STATUS_APPROVED'
              ? $t('hr.page.calendar.approved')
              : $t('hr.page.calendar.pending') }}
          </Tag>
        </div>
      </div>
    </Teleport>

    <!-- Legend -->
    <div class="tl-legend">
      <div v-for="at in absenceTypes" :key="at.id" class="tl-legend-item">
        <span class="tl-legend-dot" :style="{ backgroundColor: at.color || '#4096ff' }" />
        <span>{{ at.name }}</span>
      </div>
      <div class="tl-legend-item">
        <span class="tl-legend-dot tl-legend-dot-pending" />
        <span>{{ $t('hr.page.calendar.pending') }}</span>
      </div>
    </div>

    <RequestDrawerComponent />
  </Page>
</template>

<script lang="ts">
// Non-setup helper (deterministic color from string)
function stringToColor(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash) % 360;
  return `hsl(${hue}, 55%, 55%)`;
}
</script>

<style scoped>
/* ===== Toolbar ===== */
.tl-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0 16px;
  gap: 16px;
  flex-wrap: wrap;
}
.tl-toolbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.tl-toolbar-center {
  display: flex;
  align-items: center;
}
.tl-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.tl-nav-group {
  display: flex;
  align-items: center;
  background: var(--ant-color-bg-container, #fff);
  border: 1px solid var(--ant-color-border, #d9d9d9);
  border-radius: 8px;
  overflow: hidden;
}
.tl-nav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--ant-color-text, #333);
  transition: background 0.15s;
}
.tl-nav-btn:hover {
  background: var(--ant-color-bg-text-hover, #f5f5f5);
}
.tl-today-btn {
  height: 34px;
  padding: 0 14px;
  border: none;
  border-left: 1px solid var(--ant-color-border, #d9d9d9);
  border-right: 1px solid var(--ant-color-border, #d9d9d9);
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: var(--ant-color-text, #333);
  transition: background 0.15s;
}
.tl-today-btn:hover {
  background: var(--ant-color-bg-text-hover, #f5f5f5);
}
.tl-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--ant-color-text, #333);
  margin: 0;
  white-space: nowrap;
}
.tl-view-switcher {
  display: flex;
  background: var(--ant-color-bg-container, #fff);
  border: 1px solid var(--ant-color-border, #d9d9d9);
  border-radius: 8px;
  overflow: hidden;
}
.tl-view-btn {
  height: 34px;
  padding: 0 16px;
  border: none;
  border-right: 1px solid var(--ant-color-border, #d9d9d9);
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: var(--ant-color-text-secondary, #666);
  transition: all 0.15s;
  white-space: nowrap;
}
.tl-view-btn:last-child {
  border-right: none;
}
.tl-view-btn.active {
  background: var(--ant-color-primary, #1677ff);
  color: #fff;
}
.tl-view-btn:not(.active):hover {
  background: var(--ant-color-bg-text-hover, #f5f5f5);
}

/* ===== Timeline Container ===== */
.tl-container {
  border: 1px solid var(--ant-color-border, #d9d9d9);
  border-radius: 10px;
  overflow: auto;
  background: var(--ant-color-bg-container, #fff);
  max-height: calc(100vh - 280px);
  position: relative;
}
.tl-container.is-dragging {
  cursor: crosshair;
  user-select: none;
}
.tl-scroll {
  display: flex;
  flex-direction: column;
  min-width: 100%;
}

/* ===== Headers ===== */
.tl-header-row {
  display: flex;
  position: sticky;
  top: 0;
  z-index: 20;
  background: var(--ant-color-bg-layout, #f5f5f5);
  border-bottom: 1px solid var(--ant-color-border, #d9d9d9);
}
.tl-header-months {
  z-index: 21;
}
.tl-header-corner {
  position: sticky;
  left: 0;
  z-index: 25;
  background: var(--ant-color-bg-layout, #f5f5f5);
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-right: 1px solid var(--ant-color-border, #d9d9d9);
  flex-shrink: 0;
}
.tl-emp-header {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--ant-color-text-secondary, #888);
}
.tl-month-span {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--ant-color-text-secondary, #888);
  border-right: 1px solid var(--ant-color-border-secondary, #e8e8e8);
  padding: 6px 0;
  flex-shrink: 0;
}
.tl-day-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4px 0 6px;
  border-right: 1px solid var(--ant-color-border-secondary, #f0f0f0);
  flex-shrink: 0;
  gap: 1px;
}
.tl-day-header.weekend {
  background: rgba(0, 0, 0, 0.02);
}
.tl-day-header.today {
  background: rgba(22, 119, 255, 0.06);
}
.tl-day-name {
  font-size: 10px;
  font-weight: 500;
  color: var(--ant-color-text-quaternary, #bbb);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.tl-day-num {
  font-size: 13px;
  font-weight: 600;
  color: var(--ant-color-text, #333);
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
}
.tl-day-num.today-badge {
  background: var(--ant-color-primary, #1677ff);
  color: #fff;
}
.weekend .tl-day-name,
.weekend .tl-day-num:not(.today-badge) {
  color: var(--ant-color-text-quaternary, #ccc);
}

/* ===== Department rows ===== */
.tl-dept-row {
  display: flex;
  height: 36px;
  background: var(--ant-color-bg-layout, #f5f5f5);
  border-bottom: 1px solid var(--ant-color-border-secondary, #f0f0f0);
  cursor: pointer;
  transition: background 0.15s;
}
.tl-dept-row:hover {
  background: var(--ant-color-bg-text-hover, #eee);
}
.tl-dept-label {
  position: sticky;
  left: 0;
  z-index: 10;
  background: inherit;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 16px;
  font-size: 12px;
  font-weight: 600;
  color: var(--ant-color-text-secondary, #666);
  border-right: 1px solid var(--ant-color-border, #d9d9d9);
  flex-shrink: 0;
}
.tl-dept-chevron {
  transition: transform 0.2s;
  flex-shrink: 0;
}
.tl-dept-chevron.collapsed {
  transform: rotate(0deg);
}
.tl-dept-chevron:not(.collapsed) {
  transform: rotate(90deg);
}
.tl-dept-count {
  font-weight: 400;
  color: var(--ant-color-text-quaternary, #bbb);
}
.tl-dept-fill {
  flex: 1;
}

/* ===== User rows ===== */
.tl-emp-row {
  display: flex;
  height: 52px;
  border-bottom: 1px solid var(--ant-color-border-secondary, #f0f0f0);
  transition: background 0.1s;
}
.tl-emp-row:hover {
  background: rgba(0, 0, 0, 0.015);
}
.tl-emp-cell {
  position: sticky;
  left: 0;
  z-index: 10;
  background: var(--ant-color-bg-container, #fff);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 12px;
  border-right: 1px solid var(--ant-color-border, #d9d9d9);
  flex-shrink: 0;
}
.tl-emp-row:hover .tl-emp-cell {
  background: var(--ant-color-bg-container, #fff);
}
.tl-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
  text-transform: uppercase;
}
.tl-emp-info {
  overflow: hidden;
}
.tl-emp-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--ant-color-text, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tl-emp-title {
  font-size: 11px;
  color: var(--ant-color-text-quaternary, #bbb);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ===== Timeline cells ===== */
.tl-row-timeline {
  position: relative;
  display: flex;
  flex-shrink: 0;
}
.tl-cell {
  height: 52px;
  border-right: 1px solid var(--ant-color-border-secondary, #f0f0f0);
  flex-shrink: 0;
  cursor: crosshair;
  transition: background 0.05s;
}
.tl-cell.weekend {
  background: rgba(0, 0, 0, 0.018);
}
.tl-cell.today {
  background: rgba(22, 119, 255, 0.04);
}
.tl-cell.drag-highlight {
  background: rgba(22, 119, 255, 0.12) !important;
}

/* ===== Drag ghost ===== */
.tl-drag-ghost {
  position: absolute;
  top: 8px;
  height: 36px;
  background: rgba(22, 119, 255, 0.15);
  border: 2px dashed rgba(22, 119, 255, 0.5);
  border-radius: 8px;
  pointer-events: none;
  z-index: 5;
}

/* ===== Resize ghost ===== */
.tl-resize-ghost {
  position: absolute;
  top: 8px;
  height: 36px;
  border-radius: 8px;
  pointer-events: none;
  z-index: 4;
}

/* ===== Leave bars ===== */
.tl-bar {
  position: absolute;
  top: 8px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 6;
  transition: box-shadow 0.15s, transform 0.1s;
  overflow: hidden;

  /* Default solid bar */
  background: var(--bar-color, #4096ff);
  color: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}
.tl-bar:hover {
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.18);
  transform: translateY(-1px);
  z-index: 8;
}
.tl-bar.pending {
  background: transparent !important;
  border: 2px dashed var(--bar-color, #4096ff);
  color: var(--bar-color, #4096ff);
  box-shadow: none;
  background-image: repeating-linear-gradient(
    -45deg,
    transparent,
    transparent 4px,
    color-mix(in srgb, var(--bar-color, #4096ff) 8%, transparent) 4px,
    color-mix(in srgb, var(--bar-color, #4096ff) 8%, transparent) 8px
  ) !important;
}
.tl-bar.pending:hover {
  background-image: repeating-linear-gradient(
    -45deg,
    transparent,
    transparent 4px,
    color-mix(in srgb, var(--bar-color, #4096ff) 15%, transparent) 4px,
    color-mix(in srgb, var(--bar-color, #4096ff) 15%, transparent) 8px
  ) !important;
  transform: translateY(-1px);
}
.tl-bar-label {
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 8px;
  letter-spacing: 0.01em;
}

/* ===== Resize handles ===== */
.tl-bar-handle {
  position: absolute;
  top: 0;
  width: 8px;
  height: 100%;
  cursor: col-resize;
  z-index: 2;
}
.tl-bar-handle::after {
  content: '';
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.15s;
}
.tl-bar:hover .tl-bar-handle::after {
  opacity: 0.4;
}
.tl-bar-handle-left {
  left: 0;
  border-radius: 8px 0 0 8px;
}
.tl-bar-handle-left::after {
  left: 2px;
}
.tl-bar-handle-right {
  right: 0;
  border-radius: 0 8px 8px 0;
}
.tl-bar-handle-right::after {
  right: 2px;
}

/* ===== Today marker line ===== */
.tl-today-line {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--ant-color-primary, #1677ff);
  z-index: 7;
  pointer-events: none;
  opacity: 0.5;
}

/* ===== Tooltip ===== */
.tl-tooltip {
  position: fixed;
  z-index: 1000;
  background: var(--ant-color-bg-elevated, #fff);
  border: 1px solid var(--ant-color-border, #d9d9d9);
  border-radius: 8px;
  padding: 10px 14px;
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.12);
  transform: translate(-50%, -100%);
  pointer-events: none;
  min-width: 180px;
}
.tl-tooltip-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--ant-color-text, #333);
  margin-bottom: 4px;
}
.tl-tooltip-type {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--ant-color-text-secondary, #666);
  margin-bottom: 4px;
}
.tl-tooltip-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.tl-tooltip-dates {
  font-size: 11px;
  color: var(--ant-color-text-quaternary, #999);
  margin-bottom: 4px;
}
.tl-tooltip-meta {
  display: flex;
  align-items: center;
  font-size: 11px;
  color: var(--ant-color-text-quaternary, #999);
}

/* ===== Legend ===== */
.tl-legend {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 4px 0;
  flex-wrap: wrap;
}
.tl-legend-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--ant-color-text-secondary, #666);
}
.tl-legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
  flex-shrink: 0;
}
.tl-legend-dot-pending {
  background: transparent;
  border: 2px dashed var(--ant-color-text-quaternary, #999);
  width: 10px;
  height: 10px;
}

/* ===== Empty state ===== */
.tl-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
  font-size: 14px;
  color: var(--ant-color-text-quaternary, #999);
}
</style>
