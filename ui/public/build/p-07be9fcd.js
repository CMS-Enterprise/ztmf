import{p as e,w as a,d as n,N as i,a as l,b as t}from"./p-80f0c69a.js";export{s as setNonce}from"./p-80f0c69a.js";(()=>{e.i=a.__cssshim;const t=Array.from(n.querySelectorAll("script")).find((e=>new RegExp(`/${i}(\\.esm)?\\.js($|\\?|#)`).test(e.src)||e.getAttribute("data-stencil-namespace")===i)),s=t["data-opts"]||{};return"onbeforeload"in t&&!history.scrollRestoration?{then(){}}:(s.resourcesUrl=new URL(".",new URL(t.getAttribute("data-resources-url")||t.src,a.location.href)).href,((l,t)=>{const s=`__sc_import_${i.replace(/\s|-/g,"_")}`;try{a[s]=new Function("w",`return import(w);//${Math.random()}`)}catch(r){const i=new Map;a[s]=r=>{var c;const d=new URL(r,l).href;let o=i.get(d);if(!o){const l=n.createElement("script");l.type="module",l.crossOrigin=t.crossOrigin,l.src=URL.createObjectURL(new Blob([`import * as m from '${d}'; window.${s}.m = m;`],{type:"application/javascript"}));const r=null!==(c=e.l)&&void 0!==c?c:function(e){var a,n,i;return null!==(i=null===(n=null===(a=e.head)||void 0===a?void 0:a.querySelector('meta[name="csp-nonce"]'))||void 0===n?void 0:n.getAttribute("content"))&&void 0!==i?i:void 0}(n);null!=r&&l.setAttribute("nonce",r),o=new Promise((e=>{l.onload=()=>{e(a[s].m),l.remove()}})),i.set(d,o),n.head.appendChild(l)}return o}}})(s.resourcesUrl,t),a.customElements?l(s):__sc_import_component_library("./p-c3ec6642.js").then((()=>s)))})().then((e=>t([["p-20ff55f2",[[1,"va-omb-info",{benefitType:[1,"benefit-type"],expDate:[1,"exp-date"],ombNumber:[1,"omb-number"],resBurden:[2,"res-burden"],visible:[32]}]]],["p-873a0eb5",[[1,"va-date",{required:[4],label:[1],name:[1],hint:[1],error:[1537],monthYearOnly:[4,"month-year-only"],value:[1537],invalidDay:[1028,"invalid-day"],invalidMonth:[1028,"invalid-month"],invalidYear:[1028,"invalid-year"],enableAnalytics:[4,"enable-analytics"]}]]],["p-cd29dc9c",[[1,"va-memorable-date",{required:[4],uswds:[4],monthSelect:[4,"month-select"],label:[1],name:[1],hint:[1],error:[1537],value:[1537],invalidDay:[1028,"invalid-day"],invalidMonth:[1028,"invalid-month"],invalidYear:[1028,"invalid-year"],enableAnalytics:[4,"enable-analytics"]}]]],["p-58cf2068",[[1,"va-notification",{visible:[4],symbol:[1],closeBtnAriaLabel:[1,"close-btn-aria-label"],closeable:[516],hasBorder:[516,"has-border"],hasCloseText:[4,"has-close-text"],headline:[1],headlineLevel:[1,"headline-level"],dateTime:[1,"date-time"],href:[1],text:[1],disableAnalytics:[4,"disable-analytics"]},[[0,"component-library-analytics","handleLinkAnalytics"]]]]],["p-41b29f72",[[1,"va-banner",{disableAnalytics:[4,"disable-analytics"],showClose:[4,"show-close"],headline:[1],type:[1],visible:[4],windowSession:[4,"window-session"],dismissedBanners:[32]}]]],["p-2ccc3bc8",[[1,"va-button-pair",{continue:[4],disableAnalytics:[4,"disable-analytics"],primaryLabel:[1,"primary-label"],secondaryLabel:[1,"secondary-label"],submit:[4],update:[4],uswds:[4]},[[0,"component-library-analytics","handleButtonAnalytics"]]]]],["p-4d2a9ef3",[[1,"va-file-input",{label:[1],name:[1],buttonText:[1,"button-text"],required:[4],accept:[1],error:[1],hint:[1],enableAnalytics:[4,"enable-analytics"]}]]],["p-37bee4c8",[[1,"va-privacy-agreement",{checked:[1028],showError:[4,"show-error"],uswds:[4],enableAnalytics:[4,"enable-analytics"]}]]],["p-ac9e53cd",[[1,"va-accordion",{openSingle:[4,"open-single"],uswds:[4],disableAnalytics:[4,"disable-analytics"],sectionHeading:[1,"section-heading"],expanded:[32]},[[0,"accordionItemToggled","itemToggledHandler"]]]]],["p-c2d4ff84",[[1,"va-accordion-item",{header:[1],subheader:[1],open:[4],level:[2],bordered:[4],uswds:[4],slotHeader:[32],slotTag:[32]}]]],["p-8c7205b3",[[1,"va-additional-info",{trigger:[1],uswds:[4],disableAnalytics:[4,"disable-analytics"],disableBorder:[4,"disable-border"],open:[32]},[[9,"resize","handleResize"]]]]],["p-b89cd664",[[1,"va-alert-expandable",{status:[1],trigger:[1],disableAnalytics:[4,"disable-analytics"],iconless:[4],open:[32]},[[9,"resize","handleResize"]]]]],["p-4cca582d",[[1,"va-back-to-top",{revealed:[32],isDocked:[32]}]]],["p-9b743d2c",[[1,"va-breadcrumbs",{label:[1],uswds:[4],wrapping:[4],breadcrumbList:[8,"breadcrumb-list"],disableAnalytics:[4,"disable-analytics"],formattedBreadcrumbs:[32]}]]],["p-0e1ac897",[[1,"va-checkbox-group",{label:[1],required:[4],error:[1],enableAnalytics:[4,"enable-analytics"],hint:[1],uswds:[4]},[[0,"vaChange","vaChangeHandler"]]]]],["p-b49da884",[[1,"va-featured-content"]]],["p-1a5d1589",[[1,"va-icon",{icon:[1],size:[2],srtext:[1]}]]],["p-0723fc47",[[1,"va-loading-indicator",{message:[1],label:[1],setFocus:[4,"set-focus"],enableAnalytics:[4,"enable-analytics"]}]]],["p-5d85a53a",[[1,"va-maintenance-banner",{disableAnalytics:[4,"disable-analytics"],bannerId:[1,"banner-id"],maintenanceStartDateTime:[1,"maintenance-start-date-time"],maintenanceEndDateTime:[1,"maintenance-end-date-time"],maintenanceTitle:[1,"maintenance-title"],upcomingWarnStartDateTime:[1,"upcoming-warn-start-date-time"],upcomingWarnTitle:[1,"upcoming-warn-title"]}]]],["p-6373c7c1",[[1,"va-need-help"]]],["p-31c6dc24",[[1,"va-number-input",{label:[1],error:[1],required:[4],inputmode:[1],enableAnalytics:[4,"enable-analytics"],name:[1],min:[8],max:[8],hint:[1],messageAriaDescribedby:[1,"message-aria-describedby"],value:[1537],currency:[4],width:[1],uswds:[4]}]]],["p-0a4cce63",[[1,"va-official-gov-banner",{disableAnalytics:[4,"disable-analytics"],tld:[1]}]]],["p-218236fb",[[1,"va-on-this-page",{disableAnalytics:[4,"disable-analytics"]}]]],["p-1882de18",[[1,"va-pagination",{ariaLabelSuffix:[1,"aria-label-suffix"],enableAnalytics:[4,"enable-analytics"],maxPageListLength:[2,"max-page-list-length"],page:[2],pages:[2],showLastPage:[4,"show-last-page"],unbounded:[4],uswds:[4]}]]],["p-f24786d7",[[1,"va-process-list",{uswds:[4]}]]],["p-bcb3306b",[[4,"va-process-list-item",{header:[1],level:[2],active:[4],pending:[4],checkmark:[4]}]]],["p-1a42d8a7",[[1,"va-progress-bar",{enableAnalytics:[4,"enable-analytics"],percent:[2],label:[1]}]]],["p-c1568779",[[1,"va-promo-banner",{href:[1],type:[1],disableAnalytics:[4,"disable-analytics"],dismissedBanners:[32]}]]],["p-1f9690f8",[[1,"va-radio",{label:[1],hint:[1],required:[4],error:[1],enableAnalytics:[4,"enable-analytics"],uswds:[4],labelHeaderLevel:[1,"label-header-level"]},[[0,"keydown","handleKeyDown"],[0,"radioOptionSelected","radioOptionSelectedHandler"]]]]],["p-74ec8616",[[1,"va-radio-option",{name:[1],label:[1],value:[1],checked:[4],uswds:[4],tile:[4],description:[1],disabled:[4],ariaDescribedby:[1,"aria-describedby"]}]]],["p-42786cad",[[1,"va-search-input",{buttonText:[1,"button-text"],label:[1],suggestions:[8],value:[1537],formattedSuggestions:[32],isListboxOpen:[32]}]]],["p-d3289af8",[[1,"va-segmented-progress-bar",{enableAnalytics:[4,"enable-analytics"],current:[2],total:[2],label:[1],uswds:[4],headerLevel:[2,"header-level"],progressTerm:[1,"progress-term"],labels:[1],centeredLabels:[4,"centered-labels"],counters:[1],headingText:[1,"heading-text"]}]]],["p-b7fab2ff",[[1,"va-table",{tableTitle:[1,"table-title"],sortColumn:[2,"sort-column"],descending:[4],sortAscending:[32]}]]],["p-222c67f2",[[1,"va-table-row"]]],["p-0c1292b5",[[1,"va-textarea",{label:[1],error:[1],placeholder:[1],name:[1],required:[4],hint:[1],messageAriaDescribedby:[1,"message-aria-describedby"],maxlength:[2],value:[1537],enableAnalytics:[4,"enable-analytics"],uswds:[4],charcount:[4]}]]],["p-209e43ae",[[1,"va-alert",{status:[513],backgroundOnly:[4,"background-only"],disableAnalytics:[4,"disable-analytics"],visible:[4],closeBtnAriaLabel:[1,"close-btn-aria-label"],closeable:[516],fullWidth:[4,"full-width"],uswds:[4],slim:[4]}]]],["p-b07d1752",[[1,"va-checkbox",{label:[1],error:[513],description:[1],required:[4],enableAnalytics:[4,"enable-analytics"],checked:[1028],hint:[1],tile:[4],uswds:[516],checkboxDescription:[1,"checkbox-description"],disabled:[4],messageAriaDescribedby:[1,"message-aria-describedby"]}]]],["p-d0b2d57c",[[1,"va-modal",{clickToClose:[4,"click-to-close"],disableAnalytics:[4,"disable-analytics"],large:[516],modalTitle:[1,"modal-title"],uswds:[4],forcedModal:[4,"forced-modal"],unstyled:[4],initialFocusSelector:[1,"initial-focus-selector"],primaryButtonText:[1,"primary-button-text"],secondaryButtonText:[1,"secondary-button-text"],status:[1],visible:[516],ariaHiddenNodeExceptions:[16],shifted:[32],focusableChildren:[32]},[[0,"component-library-analytics","handleButtonClickAnalytics"],[0,"click","handleClick"],[8,"keydown","handleKeyDown"],[16,"focusin","handleFocus"]]],[1,"va-telephone",{contact:[1],extension:[2],notClickable:[4,"not-clickable"],international:[4],tty:[4],sms:[4],vanity:[1]}]]],["p-a9a81716",[[1,"va-card",{showShadow:[4,"show-shadow"]}],[1,"va-link",{abbrTitle:[1,"abbr-title"],active:[516],calendar:[4],channel:[4],disableAnalytics:[4,"disable-analytics"],download:[4],href:[1],filename:[1],filetype:[1],pages:[2],text:[1],video:[4]}]]],["p-9158ced3",[[1,"va-select",{required:[4],label:[1],name:[1],value:[1537],error:[1],reflectInputError:[4,"reflect-input-error"],invalid:[4],enableAnalytics:[4,"enable-analytics"],uswds:[516],hint:[1],options:[32]}],[1,"va-text-input",{label:[1],error:[1],reflectInputError:[4,"reflect-input-error"],invalid:[4],required:[4],inputmode:[1],type:[1],maxlength:[2],minlength:[2],autocomplete:[1],enableAnalytics:[4,"enable-analytics"],name:[1],pattern:[1],hint:[1],messageAriaDescribedby:[1,"message-aria-describedby"],value:[1537],success:[4],width:[1],uswds:[516],charcount:[4]}]]],["p-ca117054",[[1,"va-button",{back:[516],big:[516],continue:[516],disableAnalytics:[4,"disable-analytics"],disabled:[516],label:[1],primaryAlternate:[4,"primary-alternate"],secondary:[516],submit:[4],text:[1],uswds:[516]},[[0,"click","handleClickOverride"]]]]]],e)));