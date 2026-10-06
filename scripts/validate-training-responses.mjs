#!/usr/bin/env node
import { readFile } from 'node:fs/promises';
import { createResponseValidator } from './validate-api-responses.mjs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const root=resolve(fileURLToPath(new URL('..',import.meta.url)));
const fixtures=JSON.parse(await readFile(resolve(root,'test-fixtures/api/training-load.json'),'utf8'));
const validate=await createResponseValidator();
const example=JSON.parse(await readFile(resolve(root,"test-fixtures/api/training-load-response.json"),"utf8"));
validate({name:"Browser example",operationId:"getTrainingLoad",status:200,body:example});
const paths=process.argv.slice(2);
if(!paths.length) throw new Error('Provide one or both backend training response artifacts.');
let reference;
function compare(a,b,path='$') {
  if(typeof a==='number' && typeof b==='number') {if(Math.abs(a-b)>1e-6) throw new Error(`${path}: ${a} != ${b}`);return;}
  if(a===null||b===null||typeof a!=='object'||typeof b!=='object') {if(a!==b) throw new Error(`${path}: values differ`);return;}
  const ak=Object.keys(a).sort(),bk=Object.keys(b).sort();
  if(JSON.stringify(ak)!==JSON.stringify(bk)) throw new Error(`${path}: keys differ`);
  for(const key of ak) compare(a[key],b[key],`${path}.${key}`);
}
for(const path of paths) {
  const records=JSON.parse(await readFile(resolve(path),'utf8'));
  if(records.length!==fixtures.length || new Set(records.map(r=>r.name)).size!==fixtures.length) throw new Error(`${path}: missing or duplicate fixtures`);
  for(const fixture of fixtures) {
    const record=records.find(r=>r.name===fixture.name);
    if(!record || record.operationId!=='getTrainingLoad'||record.status!==200) throw new Error(`${path}: missing ${fixture.name}`);
    validate(record);
  }
  compare(example,records.find(r=>r.name===fixtures[0].name).body);
  if(reference) compare(reference,records);
  reference=records;
}
console.log(`Validated ${fixtures.length} weekly scenarios against OpenAPI${paths.length>1?' and full Go/Kotlin response parity':''}.`);
