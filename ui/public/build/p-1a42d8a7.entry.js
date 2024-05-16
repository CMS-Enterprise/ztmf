import{r,c as i,h as a,H as e}from"./p-80f0c69a.js";const s=class{constructor(a){r(this,a),this.componentLibraryAnalytics=i(this,"component-library-analytics",7),this.enableAnalytics=!1,this.percent=void 0,this.label=void 0}componentDidRender(){!this.enableAnalytics||0!==this.percent&&100!==this.percent||this.componentLibraryAnalytics.emit({componentName:"va-progress-bar",action:"change",details:{label:this.label||`${this.percent}% complete`,percent:this.percent}})}render(){const{label:r=`${this.percent.toFixed(0)}% complete`,percent:i}=this;return a(e,null,a("div",{"aria-label":r,"aria-valuemax":"100","aria-valuemin":"0","aria-valuenow":i.toFixed(0),"aria-valuetext":r,class:"progress-bar",tabindex:"0",role:"progressbar"},a("div",{class:"progress-bar-inner",style:{width:`${i}%`}})),a("span",{"aria-atomic":"true","aria-live":"polite",class:"sr-only"},i.toFixed(0),"% complete"))}};s.style=".sr-only{border:0;clip:rect(0, 0, 0, 0);-webkit-clip-path:inset(50%);clip-path:inset(50%);height:1px;margin:-1px;overflow:hidden;padding:0;position:absolute !important;width:1px;word-wrap:normal !important}div{-webkit-box-sizing:border-box;box-sizing:border-box}.progress-bar{border:2px solid var(--color-primary);border-radius:1em;display:block;height:1em;margin:1em 0;width:100%}.progress-bar-inner{background-color:var(--color-primary);content:'&nbsp;';display:block;height:100%;max-width:100%}";export{s as va_progress_bar}