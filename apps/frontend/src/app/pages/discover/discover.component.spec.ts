import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';

import { DiscoverComponent } from './discover.component';

describe('DiscoverComponent', () => {
  let component: DiscoverComponent;
  let fixture: ComponentFixture<DiscoverComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DiscoverComponent],
      providers: [provideRouter([]), provideHttpClient()]
    }).compileComponents();

    fixture = TestBed.createComponent(DiscoverComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render the discovery form', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('Listening request');
    expect(compiled.textContent).toContain('Discover albums');
  });

  it('should fill an example request', () => {
    component.useExample(component.examples[0]);
    expect(component.query).toBe(component.examples[0]);
  });
});
