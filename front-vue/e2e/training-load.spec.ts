import { test, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
const report = JSON.parse(readFileSync(new URL("../../test-fixtures/api/training-load-response.json", import.meta.url), "utf8"));

test("weekly load explains missing readings, sources and linked contributions", async ({page}) => {
  let failure=true;
  const requests: string[]=[];
  await page.route("**/api/**", async route => {
    const url=new URL(route.request().url());
    let body: unknown={};let status=200;
    if(url.pathname.endsWith("/training-load")) {
      requests.push(url.searchParams.get("week")!);
      if(failure) {body={message:"Unable to load weekly data",description:"Try again",code:500};status=500;} else body=report;
    } else if(url.pathname.endsWith("/athletes/me")) body={firstname:"Ada",lastname:"Rider",ftp:200,weight:65};
    else if(url.pathname.endsWith("/performance-settings")) body={ftpHistory:[],weightKg:null};
    else if(url.pathname.endsWith("/dashboard")) body={years:["2026"],sports:[]};
    await route.fulfill({status,contentType:"application/json",body:JSON.stringify(body)});
  });
  await page.goto("/training-load?week=2026-10-05");
  await expect(page.getByRole("heading",{name:"Training load",exact:true})).toBeVisible();
  await expect(page.getByRole("button",{name:"Retry",exact:true})).toBeVisible();
  failure=false;
  await page.getByRole("button",{name:"Retry",exact:true}).click();
  await expect(page.getByRole("heading",{name:"Activities behind the numbers"})).toBeVisible();
  await expect(page.getByText("2 activities have unknown load.",{exact:false})).toBeVisible();
  await expect(page.getByText("32.0 points",{exact:false}).first()).toBeVisible();
  await expect(page.getByRole("row").filter({hasText:"Ride 4"})).toContainText("No power readings available");
  await expect(page.getByRole("row").filter({hasText:"Ride 3"})).toContainText("0.0");
  await expect(page.getByRole("link",{name:"Ride 1",exact:true}).first()).toHaveAttribute("href","/activities/1");
  await page.getByText("How this report is calculated",{exact:true}).click();
  await expect(page.getByText("Load = covered hours",{exact:false})).toBeVisible();
  for (const close of await page.getByRole("button",{name:"Close",exact:true}).all()) await close.click();
  await page.evaluate(()=>window.scrollTo(0,0));
  await page.screenshot({path:"../test-results/training-load-desktop.png",fullPage:true});
  await page.getByRole("button",{name:"Previous week",exact:true}).click();
  await expect.poll(()=>requests.at(-1)).toBe("2026-09-28");
  await page.getByRole("button",{name:"Next week",exact:true}).click();
  await expect.poll(()=>requests.at(-1)).toBe("2026-10-05");
  await expect(page.getByRole("heading",{name:"Activities behind the numbers"})).toBeVisible();
  await page.setViewportSize({width:390,height:844});
  await page.evaluate(()=>window.scrollTo(0,0));
  await page.screenshot({path:"../test-results/training-load-mobile.png",fullPage:true});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
});

test("a ride with power gaps stays visible with separate partial load", async ({page}) => {
  const partial = structuredClone(report);
  const ride = partial.activities.find((row: {activityId: number}) => row.activityId === 1);
  ride.name = "Long ride with power gaps";
  ride.reason = "partial";
  ride.load = 33.33333333333333;
  ride.scoredSeconds = 1200;
  ride.coveredSeconds = 1200;
  ride.recordedSeconds = 1300;
  ride.elapsedSeconds = 1300;
  const week = partial.weeks[0];
  week.summary.measuredLoad = null;
  week.summary.partialMeasuredLoad = ride.load;
  week.summary.partialCount = 1;
  const day = week.days.find((value: {startDate: string}) => value.startDate === ride.date);
  day.measuredLoad = null;
  day.partialMeasuredLoad = ride.load;
  day.partialCount = 1;
  await page.route("**/api/**", async route => {
    const path = new URL(route.request().url()).pathname;
    const body = path.endsWith("/training-load") ? partial
      : path.endsWith("/athletes/me") ? {firstname:"Ada",lastname:"Rider",ftp:200,weight:65}
      : path.endsWith("/performance-settings") ? {ftpHistory:[],weightKg:null}
      : {years:["2026"],sports:[]};
    await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify(body)});
  });
  await page.goto("/training-load?week=2026-10-05");
  await expect(page.getByText("1 activity has partial load",{exact:false})).toBeVisible();
  const dailyContributions = page.locator("section.panel").filter({has:page.getByRole("heading",{name:"Daily contributions"})});
  await expect(dailyContributions.getByRole("link",{name:"Long ride with power gaps"})).toHaveAttribute("href","/activities/1");
  await expect(dailyContributions).toContainText("92% scored");
  await expect(page.getByRole("row").filter({hasText:"Long ride with power gaps"})).toContainText("Partial · 92% scored");
  await expect(page.getByRole("row").filter({hasText:"Long ride with power gaps"})).toContainText("Partial load from usable continuous power only");
});
