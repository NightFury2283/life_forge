const {test}=require('node:test');
const assert=require('node:assert/strict');
const vm=require('node:vm');
const fs=require('node:fs');
const path=require('node:path');
const code=fs.readFileSync(path.join(__dirname,'../static/planner.js'),'utf8');
const reference=Date.parse('2026-11-22T22:00:00+03:00');
class ClockDate extends Date{constructor(...args){super(...(args.length?args:[reference]));}static now(){return reference;}}
function page(){const context=vm.createContext({Date:ClockDate,Intl});vm.runInContext(code,context);return expression=>vm.runInContext(expression,context);}
for(const zone of ['UTC','America/Los_Angeles','Asia/Tokyo']){
  test(`Moscow dates remain correct when device uses ${zone}`,()=>{
    const previous=process.env.TZ;process.env.TZ=zone;
    try{const run=page();assert.equal(run(`moscowInput('2026-11-26T16:00:00Z')`),'2026-11-26T19:00');assert.equal(run(`parseMoscowInput('2026-11-26T19:00').toISOString()`),'2026-11-26T16:00:00.000Z');assert.equal(run(`moscowKey('2026-11-22T23:30:00Z')`),'2026-11-23');assert.equal(run('weekStart().toISOString()'),'2026-11-15T21:00:00.000Z');run('weekOffset=1');assert.equal(run('weekStart().toISOString()'),'2026-11-22T21:00:00.000Z');assert.match(run(`fullMoscowDate('2026-11-26T16:00:00Z')`),/четверг/);}finally{if(previous===undefined)delete process.env.TZ;else process.env.TZ=previous;}
  });
}
