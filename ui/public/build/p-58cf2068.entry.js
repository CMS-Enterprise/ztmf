import{r as i,c as o,h as t,H as a,g as e}from"./p-80f0c69a.js";import{c as n}from"./p-3d46030a.js";const s=class{constructor(t){i(this,t),this.closeEvent=o(this,"closeEvent",7),this.componentLibraryAnalytics=o(this,"component-library-analytics",7),this.visible=!0,this.symbol="none",this.closeBtnAriaLabel="Close notification",this.closeable=!1,this.hasBorder=!0,this.hasCloseText=!1,this.headline=void 0,this.headlineLevel="3",this.dateTime=void 0,this.href=void 0,this.text=void 0,this.disableAnalytics=!1}handleLinkAnalytics(i){var o,t;if("va-notification"!==i.detail.componentName&&(i.stopPropagation(),!this.disableAnalytics)){const a={componentName:"va-notification",action:"linkClick",details:{clickLabel:null===(t=null===(o=i.detail)||void 0===o?void 0:o.details)||void 0===t?void 0:t.label,type:this.symbol,headline:this.headline}};this.componentLibraryAnalytics.emit(a)}}closeHandler(i){this.closeEvent.emit(i),this.disableAnalytics||this.componentLibraryAnalytics.emit({componentName:"va-notification",action:"close",details:{type:this.symbol,headline:this.headline}})}getHeadlineLevel(){const i=parseInt(this.headlineLevel,10);return i>=1&&i<=6?`h${i}`:"h3"}render(){const{visible:i,symbol:o,headline:e,dateTime:s,href:r,text:l,closeable:c,hasBorder:h,hasCloseText:d}=this,p=this.getHeadlineLevel();if(!i)return t("div",{"aria-live":"polite"});const f=n("va-notification",o,{"has-border":h}),v=`${e} ${s}`;return t(a,null,t("va-card",{"show-shadow":"true"},t("div",{class:f},t("i",{"aria-hidden":"true",role:"img",class:o}),t("div",{class:"body",role:"presentation"},e?t(p,{part:"headline","aria-describedby":v},e):null,s?t("time",{dateTime:s},s):null,t("slot",null),r&&l?t("va-link",{active:!0,href:r,text:l}):null)),c&&t("button",{class:"va-notification-close","aria-label":this.closeBtnAriaLabel,onClick:this.closeHandler.bind(this)},t("i",{"aria-hidden":"true",class:"fas fa-times-circle",role:"presentation"}),d&&t("span",null,"CLOSE"))))}get el(){return e(this)}};s.style=':host{display:block;position:relative}:host([has-border=false])>va-card{border:none !important;-webkit-box-shadow:none !important;box-shadow:none !important}::slotted([slot=date]){display:block;color:var(--color-gray-dark);font-size:16px}va-link{display:block;margin-top:3px}:host i{font-family:"Font Awesome 5 Free";font-style:normal;font-weight:900;line-height:1;margin-right:16px}i.fa-times-circle::before{content:"\\f057";font-size:20px}i.action-required::before{content:"\\f06a";color:var(--color-secondary-dark);font-size:32px}i.update::before{content:"\\f05a";color:var(--color-primary);font-size:32px}.va-notification{display:table;width:100%;vertical-align:middle;-webkit-box-sizing:border-box;box-sizing:border-box}div.body{display:table-cell;vertical-align:middle;width:100%}h1,h2,h3,h4,h5,h6{margin:0;font-size:17px;font-family:var(--font-serif);line-height:22px;width:66%}@media (min-width: 768px){h1,h2,h3,h4,h5,h6{width:100%}}p{line-height:24px}.va-notification-close{margin:18px 12px;padding:0px;width:auto;color:var(--color-gray-dark);font-size:16px;-webkit-appearance:none;-moz-appearance:none;appearance:none;border:0px;cursor:pointer;background:transparent;display:block;position:absolute;right:0px;top:0px}.va-notification-close:hover{color:var(--color-base);background-color:transparent}.va-notification-close:active,.va-notification-close:focus{color:var(--color-base);background-color:transparent;outline:2px solid var(--color-gold-light);outline-offset:2px}.va-notification-close>i{margin-right:8px;vertical-align:middle}.va-notification-close>span{font-weight:600;margin-right:5px}';export{s as va_notification}