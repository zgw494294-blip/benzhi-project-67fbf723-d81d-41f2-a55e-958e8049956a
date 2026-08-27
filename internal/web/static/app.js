const $ = s => document.querySelector(s);
const esc = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const formObject = form => Object.fromEntries(new FormData(form));
let selected = '';

async function api(url, options) {
  const response = await fetch(url, options);
  const body = await response.json();
  if (!response.ok) {
    const error = new Error(body.error || '请求失败');
    error.data = body;
    throw error;
  }
  return body;
}

async function load() {
  const query = new URLSearchParams(formObject($('#filters')));
  const result = await api('/api/trials?' + query);
  $('#total').textContent = `命中 ${result.total} 条`;
  $('#trials').innerHTML = result.items.map(t => `<li data-id="${esc(t.trialId)}"><strong>${esc(t.trialId)}</strong> · ${esc(t.speciesName)} · ${esc(t.collectionBatch)} · ${esc(t.status)} · revision ${t.revision}${t.readOnly?' · 只读':''}</li>`).join('');
  document.querySelectorAll('#trials li').forEach(item => item.onclick = () => show(item.dataset.id));
}

function metricsPanel(m) {
  const gaps = m.gaps.map(g => `<a href="#observe-form" data-day="${g.dayIndex}" data-rep="${g.replicateIndex||''}">第${g.dayIndex}日${g.replicateIndex?'组'+g.replicateIndex:''}：${g.kind}</a>`).join('、') || '无';
  const reps = m.replicateDetails.map(x => `<tr><td>${x.replicateIndex}</td><td>${x.germinated}</td><td>${x.ratePercent.toFixed(2)}%</td><td>${x.differenceFromMean.toFixed(2)}</td><td>${x.dispersionContribution.toFixed(2)}</td></tr>`).join('');
  const curve = m.cumulativeCurve.map(x => `第${x.day}日 ${x.germinated}粒/${x.ratePercent.toFixed(2)}%`).join('；');
  const f = m.formulaInputs;
  return `<h3>指标与证据</h3><div class="grid"><div class="metric">发芽率 ${m.germinationRatePercent.toFixed(2)}%</div><div class="metric">平均发芽时间 ${m.meanGerminationTime.toFixed(2)} 日</div><div class="metric">重复组离散度 ${m.replicateStdDev.toFixed(2)}</div><div class="metric">${esc(m.thresholdConclusion)}</div></div>
    <details><summary>展开计算依据与缺口</summary><p>总种子数 ${f.totalSeeds}；有效发芽 ${f.validGerminated}；按日加权发芽数 ${f.dayWeightedGerminated}；平均发芽时间 ${f.meanGerminationTimeNumerator}/${f.meanGerminationTimeDenominator}</p><p>缺口：${gaps}</p><table><thead><tr><th>组</th><th>发芽数</th><th>发芽率</th><th>与均值差</th><th>离散度贡献</th></tr></thead><tbody>${reps}</tbody></table><p>累计曲线：${esc(curve)}</p></details>`;
}

function protocolForm(t) {
  const p = t.protocol || {};
  return `<h3>处理方案预检与锁定</h3><form id="protocol-form" class="stack">
    <label>层积天数<input name="stratificationDays" type="number" value="${p.stratificationDays??0}"><small data-error="stratificationDays"></small></label>
    <label>温度 ℃<input name="temperatureCelsius" type="number" step="0.1" value="${p.temperatureCelsius??25}"><small data-error="temperatureCelsius"></small></label>
    <label>光照<input name="lightRegime" value="${esc(p.lightRegime||'16h光照')}"><small data-error="lightRegime"></small></label>
    <label>培养基<input name="substrate" value="${esc(p.substrate||'培养皿')}"><small data-error="substrate"></small></label>
    <label>观察周期<input name="observationDays" type="number" min="1" value="${p.observationDays??3}"><small data-error="observationDays"></small></label>
    <label>阈值 %<input name="germinationThresholdPercent" type="number" step="0.01" value="${p.germinationThresholdPercent??60}"><small data-error="germinationThresholdPercent"></small></label>
    <label>操作人<input name="actor" value="研究员"></label><button type="button" id="preview">预检方案</button><div id="preview-result"></div></form>`;
}

function nextObservationDay(t) {
  for (let d=1; d<=t.protocol.observationDays; d++) if (t.observations.filter(o => o.dayIndex===d).length < t.replicateCount) return d;
  return t.protocol.observationDays;
}
function observationForm(t) {
  const day = nextObservationDay(t);
  const rows = Array.from({length:t.replicateCount},(_,i)=>`<tr data-rep="${i+1}"><td>${i+1}</td><td><input name="germ" type="number" min="0" value="0"></td><td><input name="dead" type="number" min="0" value="0"></td><td><input name="temp" type="number" step="0.1" value="25"></td><td><input name="note"></td></tr>`).join('');
  return `<h3>逐日重复组批量录入</h3><form id="observe-form"><label>试验日<input name="dayIndex" type="number" value="${day}" min="1" max="${t.protocol.observationDays}"></label><label>观察员<input name="actor" value="观察员"></label><table><thead><tr><th>重复组</th><th>新增发芽</th><th>新增失活</th><th>温度</th><th>备注</th></tr></thead><tbody>${rows}</tbody></table><button>原子提交当日全部组</button></form>`;
}

function deviationPanel(t) {
  const list = t.deviations.map(d => `<li>${esc(d.id)} · ${esc(d.kind)} · 窗口 ${d.windowStart}-${d.windowEnd} · ${d.resolved?'已销项':'未解决'}${d.resolved?'':`<form class="resolve" data-id="${esc(d.id)}"><input name="responsiblePerson" placeholder="责任人"><input name="completionDescription" placeholder="完成说明"><input name="observationIds" placeholder="观察ID，逗号分隔"><input name="actor" value="研究员"><button>证据销项</button></form>`}</li>`).join('') || '<li>无</li>';
  return `<h3>偏差与纠正证据</h3><ul>${list}</ul><form id="deviation-form"><select name="kind"><option>漏记</option><option>环境越界</option><option>样本污染</option></select><input name="description" placeholder="偏差说明"><input name="correctiveAction" placeholder="纠正措施"><input name="windowStart" type="number" min="1" placeholder="窗口起日"><input name="windowEnd" type="number" min="1" placeholder="窗口止日"><input name="actor" value="研究员"><button>登记偏差</button></form>`;
}

function correctionPanel(t) {
  if (!t.correctionItems.length) return '';
  return `<h3>退回问题清单</h3>${t.correctionItems.map(x=>`<article><strong>${esc(x.id)} · ${esc(x.status)}</strong><p>${esc(x.category)}：${esc(x.description)}；要求：${esc(x.requiredAction)}</p>${x.status==='CONFIRMED'?'':`<form class="correction" data-id="${esc(x.id)}"><input name="correctionNote" placeholder="纠正说明"><input name="referenceIds" placeholder="新业务记录 ID，逗号分隔"><label><input name="confirm" type="checkbox">确认销项</label><input name="actor" value="研究员"><button>提交材料</button></form>`}</article>`).join('')}`;
}

function reviewPanel(t, writable) {
  const p = t.reviewPackages[t.reviewPackages.length-1];
  if (!p) return '';
  const diff = p.submissionNumber>1 ? `<pre>${esc(JSON.stringify(p.difference,null,2))}</pre>` : '<p>首次提交</p>';
  let actions='';
  if(writable && t.status==='REVIEW_PENDING') actions=`<form id="approve-form"><input name="reviewer" value="审定员"><input name="conclusion" placeholder="最终结论"><button>通过审定</button></form><form id="return-form"><input name="reviewer" value="审定员"><input name="category" placeholder="问题类别"><input name="description" placeholder="问题说明"><input name="requiredAction" placeholder="要求动作"><select name="objectType"><option value="trial">试验</option><option value="protocol">方案</option><option value="metrics">指标</option><option value="observation">观察</option><option value="deviation">偏差</option></select><input name="objectId" value="${esc(t.trialId)}" placeholder="关联对象 ID"><button>结构化退回</button></form>`;
  return `<h3>审定包 #${p.submissionNumber}</h3><p>摘要 <code>${esc(p.snapshotDigest)}</code></p><p>${esc(p.metricSummary)}</p><details><summary>冻结内容与历次差异</summary><pre>${esc(JSON.stringify(p.snapshot,null,2))}</pre>${diff}</details>${actions}`;
}

function eventPanel(d) {
  const h=d.eventHistory;
  return `<h3>状态事件与摘要链</h3><p class="${h.integrity.valid?'ok':'danger'}">${h.integrity.valid?'摘要链有效，最后可信 revision '+h.integrity.lastTrustedRevision:`异常序号 ${h.integrity.firstInvalidSequence}，最后可信 revision ${h.integrity.lastTrustedRevision}：${esc(h.integrity.message)}`}</p><form id="event-filter"><input name="eventType" placeholder="操作类型"><input name="actor" placeholder="操作者"><button>筛选事件</button></form><p>命中 ${h.total} 条</p><pre>${esc(JSON.stringify(h.items,null,2))}</pre>`;
}

async function show(id, query='') {
  selected=id;
  const d=await api(`/api/trials/${encodeURIComponent(id)}/details${query}`),t=d.trial;
  let actions='';
  if(!d.readOnly){
    if(t.status==='DRAFT') actions=protocolForm(t);
    if(t.status==='PROTOCOL_LOCKED') actions='<button id="start">开始观察</button>';
    if(t.status==='OBSERVING'||t.status==='CORRECTION_REQUIRED') actions=observationForm(t)+deviationPanel(t)+correctionPanel(t)+`<button id="submit" ${d.canSubmit?'':'disabled'}>送审并固化快照</button>`;
  }
  $('#detail').innerHTML=`<h2>${esc(t.trialId)} · ${esc(t.status)}</h2><p>${esc(t.speciesName)} / ${esc(t.accessionCode)} / ${esc(t.collectionBatch)} · revision ${t.revision}${d.readOnly?' · 只读':''}</p>${metricsPanel(d.metrics)}${reviewPanel(t,!d.readOnly)}<div id="actions">${actions}</div>${eventPanel(d)}`;
  bind(t);
}

function numericProtocol(form){const b=formObject(form);['stratificationDays','temperatureCelsius','observationDays','germinationThresholdPercent'].forEach(k=>b[k]=Number(b[k]));return b}
function setIssues(form,issues=[]){form.querySelectorAll('[data-error]').forEach(x=>x.textContent='');issues.forEach(i=>{const x=form.querySelector(`[data-error="${i.field}"]`);if(x)x.textContent=i.message})}
function post(t,action,body,key=crypto.randomUUID()){return api(`/api/trials/${encodeURIComponent(t.trialId)}/${action}`,{method:'POST',headers:{'Content-Type':'application/json','Idempotency-Key':key},body:JSON.stringify({expectedRevision:t.revision,...body})})}
function reloadAfter(p,t){return p.then(()=>Promise.all([show(t.trialId),load()])).catch(e=>alert(e.message+(e.data?.remainingByGroup?'\n余量：'+JSON.stringify(e.data.remainingByGroup):'')))}

function bind(t){
  if($('#preview')) $('#preview').onclick=async()=>{const form=$('#protocol-form'),b=numericProtocol(form);try{const v=await api(`/api/trials/${encodeURIComponent(t.trialId)}/protocol-preview`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});setIssues(form,v.issues);$('#preview-result').innerHTML=v.issues.length?'<p class="danger">请修正全部字段问题</p>':`<p>${esc(v.summary)}</p><p><code>${esc(v.contentDigest)}</code></p><button type="button" id="lock">确认锁定</button>`;if($('#lock'))$('#lock').onclick=()=>reloadAfter(post(t,'protocol',{...b,contentDigest:v.contentDigest,expectedRevision:v.revision}),t)}catch(e){alert(e.message)}};
  if($('#start')) $('#start').onclick=()=>reloadAfter(post(t,'start',{actor:'研究员'}),t);
  if($('#observe-form')) $('#observe-form').onsubmit=e=>{e.preventDefault();const form=e.target,day=Number(form.dayIndex.value),actor=form.actor.value,observations=[...form.querySelectorAll('tbody tr')].map(row=>({replicateIndex:Number(row.dataset.rep),newlyGerminated:Number(row.querySelector('[name=germ]').value),newlyNonviable:Number(row.querySelector('[name=dead]').value),temperatureCelsius:Number(row.querySelector('[name=temp]').value),note:row.querySelector('[name=note]').value,recordedBy:actor}));reloadAfter(post(t,'observe',{dayIndex:day,actor,observations}),t)};
  if($('#deviation-form')) $('#deviation-form').onsubmit=e=>{e.preventDefault();const b=formObject(e.target);b.windowStart=Number(b.windowStart);b.windowEnd=Number(b.windowEnd);reloadAfter(post(t,'deviations',b),t)};
  document.querySelectorAll('.resolve').forEach(form=>form.onsubmit=e=>{e.preventDefault();const b=formObject(form);b.deviationId=form.dataset.id;b.observationIds=b.observationIds.split(',').map(x=>x.trim()).filter(Boolean);reloadAfter(post(t,'resolve',b),t)});
  document.querySelectorAll('.correction').forEach(form=>form.onsubmit=e=>{e.preventDefault();const b=formObject(form);b.issueId=form.dataset.id;b.referenceIds=b.referenceIds.split(',').map(x=>x.trim()).filter(Boolean);b.confirm=form.confirm.checked;reloadAfter(post(t,'corrections',b),t)});
  if($('#submit')) $('#submit').onclick=()=>reloadAfter(post(t,'submit',{actor:'研究员'}),t);
  const current=t.reviewPackages[t.reviewPackages.length-1];
  if($('#approve-form')) $('#approve-form').onsubmit=e=>{e.preventDefault();const b=formObject(e.target);reloadAfter(post(t,'review',{decision:'APPROVE',reviewer:b.reviewer,conclusion:b.conclusion,snapshotDigest:current.snapshotDigest}),t)};
  if($('#return-form')) $('#return-form').onsubmit=e=>{e.preventDefault();const b=formObject(e.target);reloadAfter(post(t,'review',{decision:'RETURN',reviewer:b.reviewer,conclusion:'需纠正',snapshotDigest:current.snapshotDigest,issues:[{category:b.category,description:b.description,requiredAction:b.requiredAction,objectType:b.objectType,objectId:b.objectId}]}),t)};
  if($('#event-filter')) $('#event-filter').onsubmit=e=>{e.preventDefault();show(t.trialId,'?'+new URLSearchParams(formObject(e.target)))};
}

$('#create').onsubmit=e=>{e.preventDefault();const form=e.target,b=formObject(form);b.replicateCount=Number(b.replicateCount);b.seedsPerReplicate=Number(b.seedsPerReplicate);setIssues(form);api('/api/trials',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)}).then(()=>{form.reset();load()}).catch(error=>{const c=error.data?.conflict;if(c){const x=form.querySelector(`[data-error="${c.field}"]`);if(x)x.textContent=`${c.message}：${c.trialId}（${c.status}）`}else alert(error.message)})};
$('#filters').onsubmit=e=>{e.preventDefault();load()};$('#refresh').onclick=()=>{load();if(selected)show(selected)};load();
