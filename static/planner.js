"use strict";
const MSK = "Europe/Moscow", DAY = 86400000;
const mskDate = new Intl.DateTimeFormat("ru-RU", {timeZone:MSK,day:"numeric",month:"short"});
const mskTime = new Intl.DateTimeFormat("ru-RU", {timeZone:MSK,hour:"2-digit",minute:"2-digit",hourCycle:"h23"});
let weekOffset=0, selectedCalendars=["primary"], calendarItems=[], eventItems=[], chatHistory=[], calendarRequest=0;
function moscowKey(value){const p=new Intl.DateTimeFormat("en",{timeZone:MSK,year:"numeric",month:"2-digit",day:"2-digit"}).formatToParts(new Date(value));const part=type=>p.find(v=>v.type===type).value;return `${part("year")}-${part("month")}-${part("day")}`;}
function moscowMidnight(value=Date.now()){return new Date(moscowKey(value)+"T00:00:00+03:00");}
function moscowInput(value){return moscowKey(value)+"T"+mskTime.format(new Date(value));}
function parseMoscowInput(value){return new Date(value+"+03:00");}
function fullMoscowDate(value){return new Intl.DateTimeFormat("ru-RU",{timeZone:MSK,weekday:"long",day:"numeric",month:"long",hour:"2-digit",minute:"2-digit",hourCycle:"h23"}).format(new Date(value));}
function openView(view){document.getElementById("planner-view").hidden=view!=="planner";document.getElementById("game-view").hidden=view!=="game";document.body.dataset.view=view;document.getElementById("nav-planner").classList.toggle("active",view==="planner");document.getElementById("nav-game").classList.toggle("active",view==="game");}
function configurePlanner(status){
  renderWeek(weekStart(),[]);document.getElementById("week-scroll").scrollTop=8*60;
  document.getElementById("nav-planner").onclick=()=>openView("planner");document.getElementById("nav-game").onclick=()=>openView("game");
  document.getElementById("send").disabled=!status.chat;document.getElementById("chat-input").disabled=!status.chat;
  document.getElementById("chat-status").textContent=status.chat?"●":"○";
  if(!status.chat)document.getElementById("chat-hint").textContent=status.demo?"В демо ИИ отключён — сначала войди через Google.":"Добавь ключ GigaChat в .env и перезапусти сервер.";
  if(!status.calendar){document.getElementById("calendar-message").textContent="Google Calendar доступен в обычном режиме после входа.";for(const id of ["calendar-refresh","prev-week","next-week","today-week"])document.getElementById(id).disabled=true;}
  for(const [id,action] of [["prev-week",()=>weekOffset--],["next-week",()=>weekOffset++],["today-week",()=>weekOffset=0]])document.getElementById(id).onclick=()=>{action();loadWeek();};
  document.getElementById("calendar-refresh").onclick=loadCalendars;
  for(const b of document.querySelectorAll(".suggestion"))b.onclick=()=>{document.getElementById("chat-input").value=b.textContent;document.getElementById("chat-input").focus();};
  document.getElementById("chat-form").onsubmit=sendChat;
  document.getElementById("chat-input").onkeydown=e=>{if(e.key==="Enter"&&!e.shiftKey&&!e.isComposing){e.preventDefault();document.getElementById("chat-form").requestSubmit();}};
}
function weekStart(){const day=moscowMidnight(),weekday=new Date(+day+3*3600000).getUTCDay();return new Date(+day-((weekday+6)%7)*DAY+weekOffset*7*DAY);}
async function loadCalendars(){
  const message=document.getElementById("calendar-message");message.textContent="Загружаю календари…";
  try{calendarItems=await request("/api/v1/calendar/calendars");const list=document.getElementById("calendar-list");list.replaceChildren();
    const previous=selectedCalendars.filter(id=>calendarItems.some(c=>c.id===id));selectedCalendars=previous.length?previous:calendarItems.filter(c=>c.primary).map(c=>c.id);if(!selectedCalendars.length)selectedCalendars=["primary"];
    document.getElementById("calendar-count").textContent=String(calendarItems.length);
    for(const c of calendarItems){const label=document.createElement("label"),check=document.createElement("input");check.type="checkbox";check.value=c.id;check.checked=selectedCalendars.includes(c.id);label.append(check,document.createTextNode(c.summary));list.append(label);check.onchange=()=>{const checked=[...list.querySelectorAll("input:checked")];if(checked.length>10){check.checked=false;notice("Можно выбрать до 10 календарей.");return;}selectedCalendars=checked.map(c=>c.value);loadWeek();};}
    await loadWeek();document.getElementById("week-scroll").scrollTop=8*60;
  }catch(err){message.textContent=err.message;}
}
function updateWeekLabel(start){const end=new Date(+start+6*DAY);document.getElementById("week-label").textContent=`${mskDate.format(start)} — ${mskDate.format(end)}`;}
async function loadWeek(){
  const seq=++calendarRequest,start=weekStart(),end=new Date(+start+7*DAY),message=document.getElementById("calendar-message");message.textContent="Сверяем расписание…";updateWeekLabel(start);
  try{const events=selectedCalendars.length?await request(`/api/v1/calendar/events?start=${encodeURIComponent(start.toISOString())}&end=${encodeURIComponent(end.toISOString())}&calendars=${encodeURIComponent(selectedCalendars.join(","))}`):[];
    if(seq!==calendarRequest)return;eventItems=events;renderWeek(start,events);message.textContent=events.length?`${events.length} событий · время московское · детали по нажатию`:"Здесь есть место для твоих планов.";
  }catch(err){if(seq===calendarRequest){message.textContent=err.message;renderWeek(start,[]);}}
}
function renderWeek(start,events){
  updateWeekLabel(start);const header=document.getElementById("week-days"),grid=document.getElementById("week-grid"),allDay=document.getElementById("all-day-row");header.replaceChildren(document.createElement("span"));grid.replaceChildren();allDay.replaceChildren();
  const hasAllDay=events.some(e=>e.is_all_day);if(hasAllDay){const label=document.createElement("span");label.className="all-day-title";label.textContent="ВЕСЬ ДЕНЬ";allDay.append(label);}
  const times=document.createElement("div");times.className="time-axis";for(let h=0;h<24;h++){const el=document.createElement("span");el.textContent=String(h).padStart(2,"0")+":00";times.append(el);}grid.append(times);
  for(let i=0;i<7;i++){
    const day=new Date(+start+i*DAY),next=new Date(+day+DAY),key=moscowKey(day),title=document.createElement("div");title.className="day-heading";title.textContent=new Intl.DateTimeFormat("ru-RU",{timeZone:MSK,weekday:"short",day:"numeric"}).format(day);if(key===moscowKey(Date.now()))title.classList.add("today");header.append(title);
    const column=document.createElement("div");column.className="day-column";grid.append(column);
    if(hasAllDay){const cell=document.createElement("div");cell.className="all-day-column";for(const e of events.filter(e=>e.is_all_day&&e.start_time.slice(0,10)<=key&&key<e.end_time.slice(0,10))){const b=document.createElement("button");b.className="all-day-event";b.textContent=e.title||"Без названия";b.title=e.title;b.onclick=()=>showEvent(e);cell.append(b);}allDay.append(cell);}
    const timed=events.filter(e=>!e.is_all_day&&new Date(e.start_time)<next&&new Date(e.end_time)>day).sort((a,b)=>new Date(a.start_time)-new Date(b.start_time));const placements=[];let group=[],groupEnd=0;
    function placeGroup(){const ends=[];for(const e of group){let lane=ends.findIndex(end=>end<=new Date(e.start_time));if(lane<0)lane=ends.length;ends[lane]=+new Date(e.end_time);placements.push({event:e,lane,group});}group.lanes=ends.length;}
    for(const e of timed){if(group.length&&+new Date(e.start_time)>=groupEnd){placeGroup();group=[];groupEnd=0;}group.push(e);groupEnd=Math.max(groupEnd,+new Date(e.end_time));}if(group.length)placeGroup();
    for(const p of placements){const e=p.event,b=document.createElement("button"),s=new Date(e.start_time),en=new Date(e.end_time),minute=Math.max(0,(s-day)/60000),duration=Math.max(20,(Math.min(+en,+next)-Math.max(+s,+day))/60000);
      const palette=["","color-cyan","color-pink","color-green"],index=calendarItems.findIndex(c=>c.id===e.calendar_id);b.className="calendar-event "+(e.color_id==="11"?"color-red":palette[Math.max(0,index)%palette.length]);
      b.style.top=`${minute}px`;b.style.height=`${duration}px`;b.style.left=`${p.lane/p.group.lanes*100}%`;b.style.width=`${100/p.group.lanes}%`;
      const strong=document.createElement("strong");strong.textContent=e.title||"Без названия";b.append(strong);if(duration>=35){const small=document.createElement("small");small.textContent=mskTime.format(s);b.append(small);}b.title=[e.title,`${fullMoscowDate(s)} — ${mskTime.format(en)} МСК`,e.description].filter(Boolean).join("\n");b.onclick=()=>showEvent(e);column.append(b);
    }
  }
}
function showEvent(e){const dialog=document.createElement("dialog");dialog.className="event-dialog";const h=document.createElement("h2");h.textContent=e.title;const date=document.createElement("p");date.textContent=e.is_all_day?"Событие на весь день":`${fullMoscowDate(e.start_time)} — ${mskTime.format(new Date(e.end_time))} МСК`;const description=document.createElement("p");description.textContent=e.description||"Без описания";const b=document.createElement("button");b.textContent="Понятно";b.onclick=()=>dialog.close();dialog.append(h,date,description,b);dialog.onclose=()=>dialog.remove();document.body.append(dialog);dialog.showModal();}
function addMessage(role,text){document.querySelector(".chat-welcome")?.remove();const el=document.createElement("article");el.className="chat-message "+role;const label=document.createElement("small");label.textContent=role==="user"?"ТЫ":"LIFEFORGE AI";const body=document.createElement("div");body.className="message-text";body.textContent=text;el.append(label,body);document.getElementById("messages").append(el);el.scrollIntoView({block:"nearest"});return el;}
async function sendChat(e){
  e.preventDefault();const input=document.getElementById("chat-input"),send=document.getElementById("send"),text=input.value.trim();if(!text||send.disabled)return;const history=chatHistory.slice(-10);input.value="";addMessage("user",text);send.disabled=true;send.textContent="Думаю…";const pending=addMessage("assistant","Смотрю расписание и собираю идеи…");
  try{const reply=await request("/api/v1/ai/chat",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({text,history})});pending.remove();const message=addMessage("assistant",reply.response);if(reply.warning){const warning=document.createElement("p");warning.className="draft-warning";warning.textContent=reply.warning;message.append(warning);}for(const draft of reply.drafts||[])message.append(draftCard(draft));chatHistory.push({role:"user",content:text},{role:"assistant",content:reply.response});chatHistory=chatHistory.slice(-10);message.scrollIntoView({block:"nearest"});
  }catch(err){pending.querySelector(".message-text").textContent=err.message;input.value=text;}finally{send.disabled=false;send.textContent="Отправить";}
}
function draftCard(draft){
  const form=document.createElement("form");form.className="draft-card";const caption=document.createElement("strong");caption.textContent="ПЛАН, КОТОРЫЙ РЕШАЕШЬ ТЫ";
  const title=document.createElement("input");title.value=draft.title;title.required=true;title.maxLength=200;title.setAttribute("aria-label","Название события");
  const date=document.createElement("input");date.type="datetime-local";date.required=true;date.value=moscowInput(draft.start_time);date.setAttribute("aria-label","Дата и время по Москве");
  const humanDate=document.createElement("small");humanDate.className="draft-date-label";const updateDate=()=>{humanDate.textContent=date.value?fullMoscowDate(parseMoscowInput(date.value))+" МСК":"Укажи дату";};updateDate();date.oninput=updateDate;
  const duration=document.createElement("input");duration.type="number";duration.min="0.1";duration.max="168";duration.step="0.1";duration.value=draft.duration;duration.required=true;duration.setAttribute("aria-label","Продолжительность в часах");
  const hint=document.createElement("small");hint.textContent="Дата по Москве · продолжительность в часах · основной календарь";
  const button=document.createElement("button");button.type="submit";button.textContent="Подтвердить событие";const requestID=crypto.randomUUID();form.append(caption,title,humanDate,date,duration,hint,button);
  form.onsubmit=e=>{e.preventDefault();perform(button,async()=>{await request("/api/v1/calendar/events",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({...draft,title:title.value,start_time:parseMoscowInput(date.value).toISOString(),duration:Number(duration.value),request_id:requestID})});button.textContent="В календаре";button.dataset.complete="true";for(const input of [title,date,duration])input.disabled=true;await loadWeek();if(typeof loadHero==="function")await loadHero();});};return form;
}
